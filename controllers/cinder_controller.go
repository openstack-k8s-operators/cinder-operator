/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	rabbitmqv1 "github.com/openstack-k8s-operators/infra-operator/apis/rabbitmq/v1beta1"
	keystonev1 "github.com/openstack-k8s-operators/keystone-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/configmap"
	"github.com/openstack-k8s-operators/lib-common/modules/common/endpoint"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"github.com/openstack-k8s-operators/lib-common/modules/database"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetClient -
func (r *CinderReconciler) GetClient() client.Client {
	return r.Client
}

// GetKClient -
func (r *CinderReconciler) GetKClient() kubernetes.Interface {
	return r.Kclient
}

// GetLogger -
func (r *CinderReconciler) GetLogger() logr.Logger {
	return r.Log
}

// GetScheme -
func (r *CinderReconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

// CinderReconciler reconciles a Cinder object
type CinderReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinders/finalizers,verbs=update
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderapis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderapis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderapis/finalizers,verbs=update
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderschedulers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderschedulers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderschedulers/finalizers,verbs=update
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderbackups/finalizers,verbs=update
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbdatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keystone.openstack.org,resources=keystoneapis,verbs=get;list;watch
// +kubebuilder:rbac:groups=rabbitmq.openstack.org,resources=transporturls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

// service account, role, rolebinding
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update
// service account permissions that are needed to grant permission to the above
// +kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid;privileged,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch

// Reconcile -
func (r *CinderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	_ = log.FromContext(ctx)

	// Fetch the Cinder instance
	instance := &cinderv1beta1.Cinder{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// For additional cleanup logic use finalizers. Return and don't requeue.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		r.Log,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() {
		// update the overall status condition if service is ready
		if instance.IsReady() {
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, condition.ReadyMessage)
		}

		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	// If we're not deleting this and the service object doesn't have our finalizer, add it.
	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) {
		return ctrl.Result{}, nil
	}

	//
	// initialize status
	//
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = condition.Conditions{}
		// initialize conditions used later as Status=Unknown
		cl := condition.CreateList(
			condition.UnknownCondition(condition.DBReadyCondition, condition.InitReason, condition.DBReadyInitMessage),
			condition.UnknownCondition(condition.DBSyncReadyCondition, condition.InitReason, condition.DBSyncReadyInitMessage),
			condition.UnknownCondition(cinderv1beta1.CinderRabbitMqTransportURLReadyCondition, condition.InitReason, cinderv1beta1.CinderRabbitMqTransportURLReadyInitMessage),
			condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
			condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
			condition.UnknownCondition(cinderv1beta1.CinderAPIReadyCondition, condition.InitReason, cinderv1beta1.CinderAPIReadyInitMessage),
			condition.UnknownCondition(cinderv1beta1.CinderSchedulerReadyCondition, condition.InitReason, cinderv1beta1.CinderSchedulerReadyInitMessage),
			condition.UnknownCondition(cinderv1beta1.CinderBackupReadyCondition, condition.InitReason, cinderv1beta1.CinderBackupReadyInitMessage),
			condition.UnknownCondition(cinderv1beta1.CinderVolumeReadyCondition, condition.InitReason, cinderv1beta1.CinderVolumeReadyInitMessage),
			condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
			// service account, role, rolebinding conditions
			condition.UnknownCondition(condition.ServiceAccountReadyCondition, condition.InitReason, condition.ServiceAccountReadyInitMessage),
			condition.UnknownCondition(condition.RoleReadyCondition, condition.InitReason, condition.RoleReadyInitMessage),
			condition.UnknownCondition(condition.RoleBindingReadyCondition, condition.InitReason, condition.RoleBindingReadyInitMessage),
		)

		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback e.g. in the cli
		return ctrl.Result{}, nil
	}
	if instance.Status.Hash == nil {
		instance.Status.Hash = map[string]string{}
	}
	if instance.Status.APIEndpoints == nil {
		instance.Status.APIEndpoints = map[string]map[string]string{}
	}
	if instance.Status.CinderVolumesReadyCounts == nil {
		instance.Status.CinderVolumesReadyCounts = map[string]int32{}
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, instance, helper)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CinderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// transportURLSecretFn - Watch for changes made to the secret associated with the RabbitMQ
	// TransportURL created and used by Cinder CRs.  Watch functions return a list of namespace-scoped
	// CRs that then get fed  to the reconciler.  Hence, in this case, we need to know the name of the
	// Cinder CR associated with the secret we are examining in the function.  We could parse the name
	// out of the "%s-cinder-transport" secret label, which would be faster than getting the list of
	// the Cinder CRs and trying to match on each one.  The downside there, however, is that technically
	// someone could randomly label a secret "something-cinder-transport" where "something" actually
	// matches the name of an existing Cinder CR.  In that case changes to that secret would trigger
	// reconciliation for a Cinder CR that does not need it.
	//
	// TODO: We also need a watch func to monitor for changes to the secret referenced by Cinder.Spec.Secret
	transportURLSecretFn := func(o client.Object) []reconcile.Request {
		result := []reconcile.Request{}

		// get all Cinder CRs
		cinders := &cinderv1beta1.CinderList{}
		listOpts := []client.ListOption{
			client.InNamespace(o.GetNamespace()),
		}
		if err := r.Client.List(context.Background(), cinders, listOpts...); err != nil {
			r.Log.Error(err, "Unable to retrieve Cinder CRs %v")
			return nil
		}

		for _, ownerRef := range o.GetOwnerReferences() {
			if ownerRef.Kind == "TransportURL" {
				for _, cr := range cinders.Items {
					if ownerRef.Name == fmt.Sprintf("%s-cinder-transport", cr.Name) {
						// return namespace and Name of CR
						name := client.ObjectKey{
							Namespace: o.GetNamespace(),
							Name:      cr.Name,
						}
						r.Log.Info(fmt.Sprintf("TransportURL Secret %s belongs to TransportURL belonging to Cinder CR %s", o.GetName(), cr.Name))
						result = append(result, reconcile.Request{NamespacedName: name})
					}
				}
			}
		}
		if len(result) > 0 {
			return result
		}
		return nil
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cinderv1beta1.Cinder{}).
		Owns(&mariadbv1.MariaDBDatabase{}).
		Owns(&cinderv1beta1.CinderAPI{}).
		Owns(&cinderv1beta1.CinderScheduler{}).
		Owns(&cinderv1beta1.CinderBackup{}).
		Owns(&cinderv1beta1.CinderVolume{}).
		Owns(&rabbitmqv1.TransportURL{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.ConfigMap{}).
		// Watch for TransportURL Secrets which belong to any TransportURLs created by Cinder CRs
		Watches(&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(transportURLSecretFn)).
		Complete(r)
}

func (r *CinderReconciler) reconcileDelete(ctx context.Context, instance *cinderv1beta1.Cinder, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s' delete", instance.Name))

	// remove db finalizer first
	db, err := database.GetDatabaseByName(ctx, helper, instance.Name)
	if err != nil && !k8s_errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if !k8s_errors.IsNotFound(err) {
		if err := db.DeleteFinalizer(ctx, helper); err != nil {
			return ctrl.Result{}, err
		}
	}

	// TODO: We might need to control how the sub-services (API, Backup, Scheduler and Volumes) are
	// deleted (when their parent Cinder CR is deleted) once we further develop their functionality

	// Service is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' delete successfully", instance.Name))

	return ctrl.Result{}, nil
}

func (r *CinderReconciler) reconcileInit(
	ctx context.Context,
	instance *cinderv1beta1.Cinder,
	helper *helper.Helper,
	serviceLabels map[string]string,
	serviceAnnotations map[string]string,
) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s' init", instance.Name))

	// Service account, role, binding
	rbacRules := []rbacv1.PolicyRule{
		{
			APIGroups:     []string{"security.openshift.io"},
			ResourceNames: []string{"anyuid", "privileged"},
			Resources:     []string{"securitycontextconstraints"},
			Verbs:         []string{"use"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"create", "get", "list", "watch", "update", "patch", "delete"},
		},
	}
	rbacResult, err := common_rbac.ReconcileRbac(ctx, helper, instance, rbacRules)
	if err != nil {
		return rbacResult, err
	} else if (rbacResult != ctrl.Result{}) {
		return rbacResult, nil
	}

	//
	// create service DB instance
	//
	db := database.NewDatabase(
		instance.Name,
		instance.Spec.DatabaseUser,
		instance.Spec.Secret,
		map[string]string{
			"dbName": instance.Spec.DatabaseInstance,
		},
	)
	// create or patch the DB
	ctrlResult, err := db.CreateOrPatchDB(
		ctx,
		helper,
	)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBReadyRunningMessage))
		return ctrlResult, nil
	}
	// wait for the DB to be setup
	ctrlResult, err = db.WaitForDBCreated(ctx, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBReadyErrorMessage,
			err.Error()))
		return ctrlResult, err
	}
	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBReadyRunningMessage))
		return ctrlResult, nil
	}
	// update Status.DatabaseHostname, used to config the service
	instance.Status.DatabaseHostname = db.GetDatabaseHostname()
	instance.Status.Conditions.MarkTrue(condition.DBReadyCondition, condition.DBReadyMessage)
	// create service DB - end

	//
	// run Cinder db sync
	//
	dbSyncHash := instance.Status.Hash[cinderv1beta1.DbSyncHash]
	jobDef := cinder.DbSyncJob(instance, serviceLabels, serviceAnnotations)

	dbSyncjob := job.NewJob(
		jobDef,
		cinderv1beta1.DbSyncHash,
		instance.Spec.PreserveJobs,
		time.Duration(5)*time.Second,
		dbSyncHash,
	)
	ctrlResult, err = dbSyncjob.DoJob(
		ctx,
		helper,
	)
	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBSyncReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBSyncReadyRunningMessage))
		return ctrlResult, nil
	}
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBSyncReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBSyncReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if dbSyncjob.HasChanged() {
		instance.Status.Hash[cinderv1beta1.DbSyncHash] = dbSyncjob.GetHash()
		r.Log.Info(fmt.Sprintf("Service '%s' - Job %s hash added - %s", instance.Name, jobDef.Name, instance.Status.Hash[cinderv1beta1.DbSyncHash]))
	}
	instance.Status.Conditions.MarkTrue(condition.DBSyncReadyCondition, condition.DBSyncReadyMessage)

	// when job passed, mark NetworkAttachmentsReadyCondition ready
	instance.Status.Conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)

	// run Cinder db sync - end

	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' init successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *CinderReconciler) reconcileNormal(ctx context.Context, instance *cinderv1beta1.Cinder, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s'", instance.Name))

	serviceLabels := map[string]string{
		common.AppSelector: cinder.ServiceName,
	}

	// ConfigMap
	configMapVars := make(map[string]env.Setter)

	//
	// create RabbitMQ transportURL CR and get the actual URL from the associated secret that is created
	//

	transportURL, op, err := r.transportURLCreateOrUpdate(ctx, instance, serviceLabels)

	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			cinderv1beta1.CinderRabbitMqTransportURLReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			cinderv1beta1.CinderRabbitMqTransportURLReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	if op != controllerutil.OperationResultNone {
		r.Log.Info(fmt.Sprintf("TransportURL %s successfully reconciled - operation: %s", transportURL.Name, string(op)))
	}

	instance.Status.TransportURLSecret = transportURL.Status.SecretName

	if instance.Status.TransportURLSecret == "" {
		r.Log.Info(fmt.Sprintf("Waiting for TransportURL %s secret to be created", transportURL.Name))
		instance.Status.Conditions.Set(condition.FalseCondition(
			cinderv1beta1.CinderRabbitMqTransportURLReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			cinderv1beta1.CinderRabbitMqTransportURLReadyRunningMessage))
		return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
	}

	instance.Status.Conditions.MarkTrue(cinderv1beta1.CinderRabbitMqTransportURLReadyCondition, cinderv1beta1.CinderRabbitMqTransportURLReadyMessage)

	// end transportURL

	//
	// check for required OpenStack secret holding passwords for service/admin user and add hash to the vars map
	//
	ospSecret, hash, err := secret.GetSecret(ctx, helper, instance.Spec.Secret, instance.Namespace)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.InputReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				condition.InputReadyWaitingMessage))
			return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, fmt.Errorf("OpenStack secret %s not found", instance.Spec.Secret)
		}
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.InputReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.InputReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	// Add a prefix to the var name to avoid accidental collision with other non-secret vars.
	configMapVars["secret-"+ospSecret.Name] = env.SetValue(hash)

	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)
	// run check OpenStack secret - end

	//
	// Create ConfigMaps and Secrets required as input for the Service and calculate an overall hash of hashes
	//

	//
	// create Configmap required for cinder input
	// - %-scripts configmap holding scripts to e.g. bootstrap the service
	// - %-config configmap holding minimal cinder config required to get the service up, user can add additional files to be added to the service
	// - parameters which has passwords gets added from the OpenStack secret via the init container
	//
	err = r.generateServiceConfigMaps(ctx, helper, instance, &configMapVars, serviceLabels)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	//
	// create hash over all the different input resources to identify if any those changed
	// and a restart/recreate is required.
	//
	_, hashChanged, err := r.createHashOfInputHashes(ctx, instance, configMapVars)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	} else if hashChanged {
		// Hash changed and instance status should be updated (which will be done by main defer func),
		// so we need to return and reconcile again
		return ctrl.Result{}, nil
	}
	// Create ConfigMaps and Secrets - end

	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)

	//
	// TODO check when/if Init, Update, or Upgrade should/could be skipped
	//

	// networks to attach to
	for _, netAtt := range instance.Spec.CinderAPI.NetworkAttachments {
		_, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.NetworkAttachmentsReadyWaitingMessage,
					netAtt))
				return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("network-attachment-definition %s not found", netAtt)
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
	}

	serviceAnnotations, err := nad.CreateNetworksAnnotation(instance.Namespace, instance.Spec.CinderAPI.NetworkAttachments)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.CinderAPI.NetworkAttachments, err)
	}

	// Handle service init
	ctrlResult, err := r.reconcileInit(ctx, instance, helper, serviceLabels, serviceAnnotations)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Handle service update
	ctrlResult, err = r.reconcileUpdate(ctx, instance, helper)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Handle service upgrade
	ctrlResult, err = r.reconcileUpgrade(ctx, instance, helper)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	//
	// normal reconcile tasks
	//

	// deploy cinder-api
	cinderAPI, op, err := r.apiDeploymentCreateOrUpdate(ctx, instance)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			cinderv1beta1.CinderAPIReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			cinderv1beta1.CinderAPIReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		r.Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// Mirror CinderAPI status' APIEndpoints and ReadyCount to this parent CR
	instance.Status.APIEndpoints = cinderAPI.Status.APIEndpoints
	instance.Status.ServiceIDs = cinderAPI.Status.ServiceIDs
	instance.Status.CinderAPIReadyCount = cinderAPI.Status.ReadyCount

	// Mirror CinderAPI's condition status
	c := cinderAPI.Status.Conditions.Mirror(cinderv1beta1.CinderAPIReadyCondition)
	if c != nil {
		instance.Status.Conditions.Set(c)
	}

	// TODO: These will not work without rabbit yet
	// deploy cinder-scheduler
	cinderScheduler, op, err := r.schedulerDeploymentCreateOrUpdate(ctx, instance)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			cinderv1beta1.CinderSchedulerReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			cinderv1beta1.CinderSchedulerReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		r.Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// Mirror CinderScheduler status' ReadyCount to this parent CR
	instance.Status.CinderSchedulerReadyCount = cinderScheduler.Status.ReadyCount

	// Mirror CinderScheduler's condition status
	c = cinderScheduler.Status.Conditions.Mirror(cinderv1beta1.CinderSchedulerReadyCondition)
	if c != nil {
		instance.Status.Conditions.Set(c)
	}

	// deploy cinder-backup
	cinderBackup, op, err := r.backupDeploymentCreateOrUpdate(ctx, instance)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			cinderv1beta1.CinderBackupReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			cinderv1beta1.CinderBackupReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		r.Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// Mirror CinderBackup status' ReadyCount to this parent CR
	instance.Status.CinderBackupReadyCount = cinderBackup.Status.ReadyCount

	// Mirror CinderBackup's condition status
	c = cinderBackup.Status.Conditions.Mirror(cinderv1beta1.CinderBackupReadyCondition)
	if c != nil {
		instance.Status.Conditions.Set(c)
	}

	// TODO: This requires some sort of backend and rabbit, and will not work without them
	// deploy cinder-volumes
	var volumeCondition *condition.Condition
	for name, volume := range instance.Spec.CinderVolumes {
		cinderVolume, op, err := r.volumeDeploymentCreateOrUpdate(ctx, instance, name, volume)
		if err != nil {
			instance.Status.Conditions.Set(condition.FalseCondition(
				cinderv1beta1.CinderVolumeReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				cinderv1beta1.CinderVolumeReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
		if op != controllerutil.OperationResultNone {
			r.Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
		}

		// Mirror CinderVolume status' ReadyCount to this parent CR
		// TODO: Somehow this status map can be nil here despite being initialized
		//       in the Reconcile function above
		if instance.Status.CinderVolumesReadyCounts == nil {
			instance.Status.CinderVolumesReadyCounts = map[string]int32{}
		}
		instance.Status.CinderVolumesReadyCounts[name] = cinderVolume.Status.ReadyCount

		// If this cinderVolume is not IsReady, mirror the condition to get the latest step it is in.
		// Could also check the overall ReadyCondition of the cinderVolume.
		if !cinderVolume.IsReady() {
			c = cinderVolume.Status.Conditions.Mirror(cinderv1beta1.CinderVolumeReadyCondition)
			// Get the condition with higher priority for volumeCondition.
			volumeCondition = condition.GetHigherPrioCondition(c, volumeCondition).DeepCopy()
		}
	}

	if volumeCondition != nil {
		// If there was a Status=False condition, set that as the CinderVolumeReadyCondition
		instance.Status.Conditions.Set(volumeCondition)
	} else {
		// The CinderVolumes are ready.
		// Using "condition.DeploymentReadyMessage" here because that is what gets mirrored
		// as the message for the other Cinder children when they are successfully-deployed
		instance.Status.Conditions.MarkTrue(cinderv1beta1.CinderVolumeReadyCondition, condition.DeploymentReadyMessage)
	}

	err = r.volumeCleanupDeployments(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *CinderReconciler) reconcileUpdate(ctx context.Context, instance *cinderv1beta1.Cinder, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s' update", instance.Name))

	// TODO: should have minor update tasks if required
	// - delete dbsync hash from status to rerun it?

	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' update successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *CinderReconciler) reconcileUpgrade(ctx context.Context, instance *cinderv1beta1.Cinder, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s' upgrade", instance.Name))

	// TODO: should have major version upgrade tasks
	// -delete dbsync hash from status to rerun it?

	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' upgrade successfully", instance.Name))
	return ctrl.Result{}, nil
}

// generateServiceConfigMaps - create create configmaps which hold scripts and service configuration
// TODO add DefaultConfigOverwrite
func (r *CinderReconciler) generateServiceConfigMaps(
	ctx context.Context,
	h *helper.Helper,
	instance *cinderv1beta1.Cinder,
	envVars *map[string]env.Setter,
	serviceLabels map[string]string,
) error {
	//
	// create Configmap/Secret required for cinder input
	// - %-scripts configmap holding scripts to e.g. bootstrap the service
	// - %-config configmap holding minimal cinder config required to get the service up, user can add additional files to be added to the service
	// - parameters which has passwords gets added from the ospSecret via the init container
	//

	cmLabels := labels.GetLabels(instance, labels.GetGroupLabel(cinder.ServiceName), serviceLabels)

	// customData hold any customization for the service.
	// custom.conf is going to /etc/<service>/<service>.conf.d
	// all other files get placed into /etc/<service> to allow overwrite of e.g. policy.json
	// TODO: make sure custom.conf can not be overwritten
	customData := map[string]string{common.CustomServiceConfigFileName: instance.Spec.CustomServiceConfig}

	for key, data := range instance.Spec.DefaultConfigOverwrite {
		customData[key] = data
	}

	keystoneAPI, err := keystonev1.GetKeystoneAPI(ctx, h, instance.Namespace, map[string]string{})
	if err != nil {
		return err
	}
	keystoneInternalURL, err := keystoneAPI.GetEndpoint(endpoint.EndpointInternal)
	if err != nil {
		return err
	}
	keystonePublicURL, err := keystoneAPI.GetEndpoint(endpoint.EndpointPublic)
	if err != nil {
		return err
	}
	templateParameters := make(map[string]interface{})
	templateParameters["ServiceUser"] = instance.Spec.ServiceUser
	templateParameters["KeystoneInternalURL"] = keystoneInternalURL
	templateParameters["KeystonePublicURL"] = keystonePublicURL

	cms := []util.Template{
		// ScriptsConfigMap
		{
			Name:               fmt.Sprintf("%s-scripts", instance.Name),
			Namespace:          instance.Namespace,
			Type:               util.TemplateTypeScripts,
			InstanceType:       instance.Kind,
			AdditionalTemplate: map[string]string{"common.sh": "/common/common.sh"},
			Labels:             cmLabels,
		},
		// ConfigMap
		{
			Name:          fmt.Sprintf("%s-config-data", instance.Name),
			Namespace:     instance.Namespace,
			Type:          util.TemplateTypeConfig,
			InstanceType:  instance.Kind,
			CustomData:    customData,
			ConfigOptions: templateParameters,
			Labels:        cmLabels,
		},
	}

	return configmap.EnsureConfigMaps(ctx, h, instance, cms, envVars)
}

// createHashOfInputHashes - creates a hash of hashes which gets added to the resources which requires a restart
// if any of the input resources change, like configs, passwords, ...
//
// returns the hash, whether the hash changed (as a bool) and any error
func (r *CinderReconciler) createHashOfInputHashes(
	ctx context.Context,
	instance *cinderv1beta1.Cinder,
	envVars map[string]env.Setter,
) (string, bool, error) {
	var hashMap map[string]string
	changed := false
	mergedMapVars := env.MergeEnvs([]corev1.EnvVar{}, envVars)
	hash, err := util.ObjectHash(mergedMapVars)
	if err != nil {
		return hash, changed, err
	}
	if hashMap, changed = util.SetHash(instance.Status.Hash, common.InputHashName, hash); changed {
		instance.Status.Hash = hashMap
		r.Log.Info(fmt.Sprintf("Input maps hash %s - %s", common.InputHashName, hash))
	}
	return hash, changed, nil
}

func (r *CinderReconciler) transportURLCreateOrUpdate(
	ctx context.Context,
	instance *cinderv1beta1.Cinder,
	serviceLabels map[string]string,
) (*rabbitmqv1.TransportURL, controllerutil.OperationResult, error) {
	transportURL := &rabbitmqv1.TransportURL{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-cinder-transport", instance.Name),
			Namespace: instance.Namespace,
			Labels:    serviceLabels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, transportURL, func() error {
		transportURL.Spec.RabbitmqClusterName = instance.Spec.RabbitMqClusterName

		err := controllerutil.SetControllerReference(instance, transportURL, r.Scheme)
		return err
	})

	return transportURL, op, err
}

func (r *CinderReconciler) apiDeploymentCreateOrUpdate(ctx context.Context, instance *cinderv1beta1.Cinder) (*cinderv1beta1.CinderAPI, controllerutil.OperationResult, error) {

	cinderAPISpec := cinderv1beta1.CinderAPISpec{
		CinderTemplate:     instance.Spec.CinderTemplate,
		CinderAPITemplate:  instance.Spec.CinderAPI,
		ExtraMounts:        instance.Spec.ExtraMounts,
		DatabaseHostname:   instance.Status.DatabaseHostname,
		TransportURLSecret: instance.Status.TransportURLSecret,
		ServiceAccount:     instance.RbacResourceName(),
	}

	deployment := &cinderv1beta1.CinderAPI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-api", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = cinderAPISpec
		if len(deployment.Spec.NodeSelector) == 0 {
			deployment.Spec.NodeSelector = instance.Spec.NodeSelector
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return deployment, op, err
}

func (r *CinderReconciler) schedulerDeploymentCreateOrUpdate(ctx context.Context, instance *cinderv1beta1.Cinder) (*cinderv1beta1.CinderScheduler, controllerutil.OperationResult, error) {

	cinderSchedulerSpec := cinderv1beta1.CinderSchedulerSpec{
		CinderTemplate:          instance.Spec.CinderTemplate,
		CinderSchedulerTemplate: instance.Spec.CinderScheduler,
		ExtraMounts:             instance.Spec.ExtraMounts,
		DatabaseHostname:        instance.Status.DatabaseHostname,
		TransportURLSecret:      instance.Status.TransportURLSecret,
		ServiceAccount:          instance.RbacResourceName(),
	}

	deployment := &cinderv1beta1.CinderScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-scheduler", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = cinderSchedulerSpec
		if len(deployment.Spec.NodeSelector) == 0 {
			deployment.Spec.NodeSelector = instance.Spec.NodeSelector
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return deployment, op, err
}

func (r *CinderReconciler) backupDeploymentCreateOrUpdate(ctx context.Context, instance *cinderv1beta1.Cinder) (*cinderv1beta1.CinderBackup, controllerutil.OperationResult, error) {

	cinderBackupSpec := cinderv1beta1.CinderBackupSpec{
		CinderTemplate:       instance.Spec.CinderTemplate,
		CinderBackupTemplate: instance.Spec.CinderBackup,
		ExtraMounts:          instance.Spec.ExtraMounts,
		DatabaseHostname:     instance.Status.DatabaseHostname,
		TransportURLSecret:   instance.Status.TransportURLSecret,
		ServiceAccount:       instance.RbacResourceName(),
	}
	deployment := &cinderv1beta1.CinderBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-backup", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = cinderBackupSpec
		if len(deployment.Spec.NodeSelector) == 0 {
			deployment.Spec.NodeSelector = instance.Spec.NodeSelector
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return deployment, op, err
}

func (r *CinderReconciler) volumeDeploymentCreateOrUpdate(ctx context.Context, instance *cinderv1beta1.Cinder, name string, volTemplate cinderv1beta1.CinderVolumeTemplate) (*cinderv1beta1.CinderVolume, controllerutil.OperationResult, error) {

	cinderVolumeSpec := cinderv1beta1.CinderVolumeSpec{
		CinderTemplate:       instance.Spec.CinderTemplate,
		CinderVolumeTemplate: volTemplate,
		ExtraMounts:          instance.Spec.ExtraMounts,
		DatabaseHostname:     instance.Status.DatabaseHostname,
		TransportURLSecret:   instance.Status.TransportURLSecret,
		ServiceAccount:       instance.RbacResourceName(),
	}

	deployment := &cinderv1beta1.CinderVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-volume-%s", instance.Name, name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = cinderVolumeSpec

		// If NodeSelector is not specified in volumeTemplate, the current
		// cinder-volume instance inherits the value from the top-level CR
		if len(volTemplate.NodeSelector) == 0 {
			deployment.Spec.NodeSelector = instance.Spec.NodeSelector
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return deployment, op, err
}

// volumeCleanupDeployments - Delete volume deployments when the volume no
// longer appears in the spec. These will be volumes named something like
// "cinder-volume-X" where "X" is not in the CinderVolumes spec.
func (r *CinderReconciler) volumeCleanupDeployments(ctx context.Context, instance *cinderv1beta1.Cinder) error {
	// Generate a list of volume CRs
	volumes := &cinderv1beta1.CinderVolumeList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.Namespace),
	}
	if err := r.Client.List(ctx, volumes, listOpts...); err != nil {
		r.Log.Error(err, "Unable to retrieve volume CRs %v")
		return nil
	}

	prefixLen := len(instance.Name + "-volume-")
	for _, volume := range volumes.Items {
		// Skip volumes that we don't own
		if cinder.GetOwningCinderName(&volume) != instance.Name {
			continue
		}

		// specName is the volume's name as it would appear in the CinderVolumes spec
		specName := volume.Name[prefixLen:]

		// Delete the volume if it's no longer in the spec
		_, exists := instance.Spec.CinderVolumes[specName]
		if !exists && volume.DeletionTimestamp.IsZero() {
			err := r.Client.Delete(ctx, &volume)
			if err != nil && !k8s_errors.IsNotFound(err) {
				err = fmt.Errorf("Error cleaning up %s: %w", volume.Name, err)
				return err
			}
		}
	}

	return nil
}
