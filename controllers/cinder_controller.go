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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	memcachedv1 "github.com/openstack-k8s-operators/infra-operator/apis/memcached/v1beta1"
	rabbitmqv1 "github.com/openstack-k8s-operators/infra-operator/apis/rabbitmq/v1beta1"
	keystonev1 "github.com/openstack-k8s-operators/keystone-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	cronjob "github.com/openstack-k8s-operators/lib-common/modules/common/cronjob"
	"github.com/openstack-k8s-operators/lib-common/modules/common/endpoint"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/service"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
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

// GetScheme -
func (r *CinderReconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

// CinderReconciler reconciles a Cinder object
type CinderReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
}

// GetLogger returns a logger object with a logging prefix of "controller.name" and additional controller context fields
func (r *CinderReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("Cinder")
}

// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinders/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderapis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderapis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderapis/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderschedulers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderschedulers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderschedulers/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderbackups/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbdatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbaccounts/finalizers,verbs=update;patch
// +kubebuilder:rbac:groups=memcached.openstack.org,resources=memcacheds,verbs=get;list;watch;
// +kubebuilder:rbac:groups=keystone.openstack.org,resources=keystoneapis,verbs=get;list;watch
// +kubebuilder:rbac:groups=rabbitmq.openstack.org,resources=transporturls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete;

// service account, role, rolebinding
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch
// service account permissions that are needed to grant permission to the above
// +kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid;privileged,resources=securitycontextconstraints,verbs=use
// +kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch

// Reconcile -
func (r *CinderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

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
		Log.Error(err, fmt.Sprintf("could not fetch Cinder instance %s", instance.Name))
		return ctrl.Result{}, err
	}

	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		Log,
	)
	if err != nil {
		Log.Error(err, fmt.Sprintf("could not instantiate helper for instance %s", instance.Name))
		return ctrl.Result{}, err
	}

	//
	// initialize status
	//
	isNewInstance := instance.Status.Conditions == nil
	if isNewInstance {
		instance.Status.Conditions = condition.Conditions{}
	}

	// Save a copy of the condtions so that we can restore the LastTransitionTime
	// when a condition's state doesn't change.
	savedConditions := instance.Status.Conditions.DeepCopy()

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() {
		// Don't update the status, if reconciler Panics
		if r := recover(); r != nil {
			Log.Info(fmt.Sprintf("panic during reconcile %v\n", r))
			panic(r)
		}
		condition.RestoreLastTransitionTimes(&instance.Status.Conditions, savedConditions)
		if instance.Status.Conditions.IsUnknown(condition.ReadyCondition) {
			instance.Status.Conditions.Set(
				instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	// Always initialize conditions used later as Status=Unknown
	cl := condition.CreateList(
		condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage),
		condition.UnknownCondition(condition.DBReadyCondition, condition.InitReason, condition.DBReadyInitMessage),
		condition.UnknownCondition(condition.DBSyncReadyCondition, condition.InitReason, condition.DBSyncReadyInitMessage),
		condition.UnknownCondition(condition.RabbitMqTransportURLReadyCondition, condition.InitReason, condition.RabbitMqTransportURLReadyInitMessage),
		condition.UnknownCondition(condition.MemcachedReadyCondition, condition.InitReason, condition.MemcachedReadyInitMessage),
		condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
		condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
		condition.UnknownCondition(cinderv1beta1.CinderAPIReadyCondition, condition.InitReason, cinderv1beta1.CinderAPIReadyInitMessage),
		condition.UnknownCondition(cinderv1beta1.CinderSchedulerReadyCondition, condition.InitReason, cinderv1beta1.CinderSchedulerReadyInitMessage),
		condition.UnknownCondition(cinderv1beta1.CinderBackupReadyCondition, condition.InitReason, cinderv1beta1.CinderBackupReadyInitMessage),
		condition.UnknownCondition(cinderv1beta1.CinderVolumeReadyCondition, condition.InitReason, cinderv1beta1.CinderVolumeReadyInitMessage),
		condition.UnknownCondition(condition.CronJobReadyCondition, condition.InitReason, condition.CronJobReadyInitMessage),
		condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
		// service account, role, rolebinding conditions
		condition.UnknownCondition(condition.ServiceAccountReadyCondition, condition.InitReason, condition.ServiceAccountReadyInitMessage),
		condition.UnknownCondition(condition.RoleReadyCondition, condition.InitReason, condition.RoleReadyInitMessage),
		condition.UnknownCondition(condition.RoleBindingReadyCondition, condition.InitReason, condition.RoleBindingReadyInitMessage),
	)

	if instance.Spec.NotificationBusInstance != nil {
		c := condition.UnknownCondition(
			condition.NotificationBusInstanceReadyCondition,
			condition.InitReason,
			condition.NotificationBusInstanceReadyInitMessage)
		cl.Set(c)
	}

	instance.Status.Conditions.Init(&cl)
	// Always mark the Generation as observed early on
	instance.Status.ObservedGeneration = instance.Generation

	// If we're not deleting this and the service object doesn't have our finalizer, add it.
	if (instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer())) || isNewInstance {
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

// fields to index to reconcile when change
const (
	passwordSecretField     = ".spec.secret"
	caBundleSecretNameField = ".spec.tls.caBundleSecretName"
	tlsAPIInternalField     = ".spec.tls.api.internal.secretName"
	tlsAPIPublicField       = ".spec.tls.api.public.secretName"
	topologyField           = ".spec.topologyRef.Name"
)

var (
	commonWatchFields = []string{
		passwordSecretField,
		caBundleSecretNameField,
		topologyField,
	}
	cinderAPIWatchFields = []string{
		passwordSecretField,
		caBundleSecretNameField,
		tlsAPIInternalField,
		tlsAPIPublicField,
		topologyField,
	}
)

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
	transportURLSecretFn := func(ctx context.Context, o client.Object) []reconcile.Request {
		result := []reconcile.Request{}

		Log := r.GetLogger(ctx)

		// get all Cinder CRs
		cinders := &cinderv1beta1.CinderList{}
		listOpts := []client.ListOption{
			client.InNamespace(o.GetNamespace()),
		}
		if err := r.Client.List(ctx, cinders, listOpts...); err != nil {
			Log.Error(err, "Unable to retrieve Cinder CRs %v")
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
						Log.Info(fmt.Sprintf("TransportURL Secret %s belongs to TransportURL belonging to Cinder CR %s", o.GetName(), cr.Name))
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

	memcachedFn := func(ctx context.Context, o client.Object) []reconcile.Request {
		Log := r.GetLogger(ctx)

		result := []reconcile.Request{}

		// get all Cinder CRs
		cinders := &cinderv1beta1.CinderList{}
		listOpts := []client.ListOption{
			client.InNamespace(o.GetNamespace()),
		}
		if err := r.Client.List(ctx, cinders, listOpts...); err != nil {
			Log.Error(err, "Unable to retrieve Cinder CRs %w")
			return nil
		}

		for _, cr := range cinders.Items {
			if o.GetName() == cr.Spec.MemcachedInstance {
				name := client.ObjectKey{
					Namespace: o.GetNamespace(),
					Name:      cr.Name,
				}
				Log.Info(fmt.Sprintf("Memcached %s is used by Cinder CR %s", o.GetName(), cr.Name))
				result = append(result, reconcile.Request{NamespacedName: name})
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
		Owns(&mariadbv1.MariaDBAccount{}).
		Owns(&cinderv1beta1.CinderAPI{}).
		Owns(&cinderv1beta1.CinderScheduler{}).
		Owns(&cinderv1beta1.CinderBackup{}).
		Owns(&cinderv1beta1.CinderVolume{}).
		Owns(&rabbitmqv1.TransportURL{}).
		Owns(&batchv1.Job{}).
		Owns(&batchv1.CronJob{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		// Watch for TransportURL Secrets which belong to any TransportURLs created by Cinder CRs
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(transportURLSecretFn)).
		Watches(&memcachedv1.Memcached{},
			handler.EnqueueRequestsFromMapFunc(memcachedFn)).
		Watches(&keystonev1.KeystoneAPI{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectForSrc),
			builder.WithPredicates(keystonev1.KeystoneAPIStatusChangedPredicate)).
		Complete(r)
}

func (r *CinderReconciler) findObjectForSrc(ctx context.Context, src client.Object) []reconcile.Request {
	requests := []reconcile.Request{}

	l := log.FromContext(ctx).WithName("Controllers").WithName("Cinder")

	crList := &cinderv1beta1.CinderList{}
	listOps := &client.ListOptions{
		Namespace: src.GetNamespace(),
	}
	err := r.Client.List(ctx, crList, listOps)
	if err != nil {
		l.Error(err, fmt.Sprintf("listing %s for namespace: %s", crList.GroupVersionKind().Kind, src.GetNamespace()))
		return requests
	}

	for _, item := range crList.Items {
		l.Info(fmt.Sprintf("input source %s changed, reconcile: %s - %s", src.GetName(), item.GetName(), item.GetNamespace()))

		requests = append(requests,
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			},
		)
	}

	return requests
}

func (r *CinderReconciler) reconcileDelete(ctx context.Context, instance *cinderv1beta1.Cinder, helper *helper.Helper) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)

	Log.Info(fmt.Sprintf("Reconciling Service '%s' delete", instance.Name))

	// remove db finalizer first
	db, err := mariadbv1.GetDatabaseByNameAndAccount(ctx, helper, cinder.DatabaseName, instance.Spec.DatabaseAccount, instance.Namespace)
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
	Log.Info(fmt.Sprintf("Reconciled Service '%s' delete successfully", instance.Name))

	return ctrl.Result{}, nil
}

func (r *CinderReconciler) reconcileInit(
	ctx context.Context,
	instance *cinderv1beta1.Cinder,
	helper *helper.Helper,
	serviceLabels map[string]string,
	serviceAnnotations map[string]string,
) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)

	Log.Info(fmt.Sprintf("Reconciling Service '%s' init", instance.Name))

	//
	// run Cinder db sync
	//
	dbSyncHash := instance.Status.Hash[cinderv1beta1.DbSyncHash]
	jobDef := cinder.DbSyncJob(instance, serviceLabels, serviceAnnotations)

	dbSyncjob := job.NewJob(
		jobDef,
		cinderv1beta1.DbSyncHash,
		instance.Spec.PreserveJobs,
		cinder.ShortDuration,
		dbSyncHash,
	)
	ctrlResult, err := dbSyncjob.DoJob(
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
		Log.Info(fmt.Sprintf("Service '%s' - Job %s hash added - %s", instance.Name, jobDef.Name, instance.Status.Hash[cinderv1beta1.DbSyncHash]))
	}
	instance.Status.Conditions.MarkTrue(condition.DBSyncReadyCondition, condition.DBSyncReadyMessage)

	// run Cinder db sync - end

	Log.Info(fmt.Sprintf("Reconciled Service '%s' init successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *CinderReconciler) reconcileNormal(ctx context.Context, instance *cinderv1beta1.Cinder, helper *helper.Helper) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)

	Log.Info(fmt.Sprintf("Reconciling Service '%s'", instance.Name))

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

	serviceLabels := map[string]string{
		common.AppSelector: cinder.ServiceName,
	}

	configVars := make(map[string]env.Setter)

	//
	// create RabbitMQ transportURL CR and get the actual URL from the associated secret that is created
	//

	transportURL, op, err := r.transportURLCreateOrUpdate(ctx, instance, serviceLabels, "")
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.RabbitMqTransportURLReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.RabbitMqTransportURLReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	if op != controllerutil.OperationResultNone {
		Log.Info(fmt.Sprintf("TransportURL %s successfully reconciled - operation: %s", transportURL.Name, string(op)))
	}

	instance.Status.TransportURLSecret = transportURL.Status.SecretName

	if instance.Status.TransportURLSecret == "" {
		Log.Info(fmt.Sprintf("Waiting for TransportURL %s secret to be created", transportURL.Name))
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.RabbitMqTransportURLReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.RabbitMqTransportURLReadyRunningMessage))
		return cinder.ResultRequeue, nil
	}

	instance.Status.Conditions.MarkTrue(condition.RabbitMqTransportURLReadyCondition, condition.RabbitMqTransportURLReadyMessage)

	// end transportURL

	//
	// create NotificationBus transportURL CR and get the actual URL from the
	// associated secret that is created
	//

	// Request TransportURL when the parameter is provided in the CR
	// and it does not match with the existing RabbitMqClusterName
	if instance.Spec.NotificationBusInstance != nil {
		// init .Status.NotificationURLSecret
		instance.Status.NotificationURLSecret = ptr.To("")

		// setting notificationBusName to an empty string ensures that we do not
		// request a new transportURL unless the two spec fields do not match
		var notificationBusName string
		if *instance.Spec.NotificationBusInstance != instance.Spec.RabbitMqClusterName {
			notificationBusName = *instance.Spec.NotificationBusInstance
		}
		notificationBusInstanceURL, op, err := r.transportURLCreateOrUpdate(ctx, instance, serviceLabels, notificationBusName)
		if err != nil {
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NotificationBusInstanceReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NotificationBusInstanceReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}

		if op != controllerutil.OperationResultNone {
			Log.Info(fmt.Sprintf("NotificationBusInstanceURL %s successfully reconciled - operation: %s", notificationBusInstanceURL.Name, string(op)))
		}

		*instance.Status.NotificationURLSecret = notificationBusInstanceURL.Status.SecretName

		if instance.Status.NotificationURLSecret == nil {
			Log.Info(fmt.Sprintf("Waiting for NotificationBusInstanceURL %s secret to be created", transportURL.Name))
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NotificationBusInstanceReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				condition.NotificationBusInstanceReadyRunningMessage))
			return cinder.ResultRequeue, nil
		}

		instance.Status.Conditions.MarkTrue(condition.NotificationBusInstanceReadyCondition, condition.NotificationBusInstanceReadyMessage)
	} else {
		// make sure we do not have an entry in the status if
		// .Spec.NotificationURLSecret is not provided
		instance.Status.NotificationURLSecret = nil
	}

	// end notificationBusInstanceURL

	//
	// Check for required memcached used for caching
	//
	memcached, err := memcachedv1.GetMemcachedByName(ctx, helper, instance.Spec.MemcachedInstance, instance.Namespace)
	if err != nil {
		Log.Info(fmt.Sprintf("%s... requeueing", condition.MemcachedReadyWaitingMessage))
		if k8s_errors.IsNotFound(err) {
			Log.Info(fmt.Sprintf("memcached %s not found", instance.Spec.MemcachedInstance))
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.MemcachedReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				condition.MemcachedReadyWaitingMessage))
			return cinder.ResultRequeue, nil
		}
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.MemcachedReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.MemcachedReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	if !memcached.IsReady() {
		Log.Info(fmt.Sprintf("%s... requeueing", condition.MemcachedReadyWaitingMessage))
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.MemcachedReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.MemcachedReadyWaitingMessage))
		return cinder.ResultRequeue, nil
	}
	// Mark the Memcached Service as Ready if we get to this point with no errors
	instance.Status.Conditions.MarkTrue(
		condition.MemcachedReadyCondition, condition.MemcachedReadyMessage)
	// run check memcached - end

	//
	// check for required OpenStack secret holding passwords for service/admin user and add hash to the vars map
	//

	result, err := verifyServiceSecret(
		ctx,
		types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.Secret},
		[]string{
			instance.Spec.PasswordSelectors.Service,
		},
		helper.GetClient(),
		&instance.Status.Conditions,
		cinder.NormalDuration,
		&configVars,
	)
	if err != nil {
		return result, err
	} else if (result != ctrl.Result{}) {
		return result, nil
	}
	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)
	// run check OpenStack secret - end

	db, result, err := r.ensureDB(ctx, helper, instance)
	if err != nil {
		return ctrl.Result{}, err
	} else if (result != ctrl.Result{}) {
		return result, nil
	}

	//
	// Create Secrets required as input for the Service and calculate an overall hash of hashes
	//
	err = r.generateServiceConfigs(ctx, helper, instance, &configVars, serviceLabels, memcached, db)
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
	_, hashChanged, err := r.createHashOfInputHashes(ctx, instance, configVars)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	} else if hashChanged {
		Log.Info(fmt.Sprintf("%s... requeueing", condition.ServiceConfigReadyInitMessage))
		instance.Status.Conditions.MarkFalse(
			condition.ServiceConfigReadyCondition,
			condition.InitReason,
			condition.SeverityInfo,
			condition.ServiceConfigReadyInitMessage)
		// Hash changed and instance status should be updated (which will be done by main defer func),
		// so we need to return and reconcile again
		return ctrl.Result{}, nil
	}

	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)

	//
	// TODO check when/if Init, Update, or Upgrade should/could be skipped
	//

	// Check networks that the DBSync job will use in reconcileInit. The ones from the API service are always enough,
	// it doesn't need the storage specific ones that volume or backup may have.
	nadList := []networkv1.NetworkAttachmentDefinition{}
	for _, netAtt := range instance.Spec.CinderAPI.NetworkAttachments {
		nad, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				Log.Info(fmt.Sprintf("network-attachment-definition %s not found", netAtt))
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.NetworkAttachmentsReadyWaitingMessage,
					netAtt))
				return cinder.ResultRequeue, fmt.Errorf(condition.NetworkAttachmentsReadyWaitingMessage, netAtt)
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}

		if nad != nil {
			nadList = append(nadList, *nad)
		}
	}

	instance.Status.Conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)

	serviceAnnotations, err := nad.EnsureNetworksAnnotation(nadList)
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

	//
	// normal reconcile tasks
	//

	// deploy cinder-api
	cinderAPI, op, err := r.apiDeploymentCreateOrUpdate(ctx, instance, memcached)
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
		Log.Info(fmt.Sprintf("API CR for %s successfully %s", instance.Name, string(op)))
	}

	// Mirror values when the data in the StatefulSet is for the current generation
	if cinderAPI.Generation == cinderAPI.Status.ObservedGeneration {
		// Mirror CinderAPI status' APIEndpoints and ReadyCount to this parent CR
		instance.Status.APIEndpoints = cinderAPI.Status.APIEndpoints
		instance.Status.ServiceIDs = cinderAPI.Status.ServiceIDs
		instance.Status.CinderAPIReadyCount = cinderAPI.Status.ReadyCount

		// Mirror CinderAPI's condition status
		c := cinderAPI.Status.Conditions.Mirror(cinderv1beta1.CinderAPIReadyCondition)
		if c != nil {
			instance.Status.Conditions.Set(c)
		}
	}

	// deploy cinder-scheduler
	cinderScheduler, op, err := r.schedulerDeploymentCreateOrUpdate(ctx, instance, memcached)
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
		Log.Info(fmt.Sprintf("Scheduler CR for %s successfully %s", instance.Name, string(op)))
	}

	// Mirror values when the data in the StatefulSet is for the current generation
	if cinderScheduler.Generation == cinderScheduler.Status.ObservedGeneration {
		// Mirror CinderScheduler status' ReadyCount to this parent CR
		instance.Status.CinderSchedulerReadyCount = cinderScheduler.Status.ReadyCount

		// Mirror CinderScheduler's condition status
		c := cinderScheduler.Status.Conditions.Mirror(cinderv1beta1.CinderSchedulerReadyCondition)
		if c != nil {
			instance.Status.Conditions.Set(c)
		}
	}

	// deploy cinder-backup, but only if necessary
	//
	// Many OpenStack deployments don't use the cinder-backup service (it's optional),
	// so there's no need to deploy it unless it's required.
	var backupCondition *condition.Condition
	if *instance.Spec.CinderBackup.Replicas > 0 {
		cinderBackup, op, err := r.backupDeploymentCreateOrUpdate(ctx, instance, memcached)
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
			Log.Info(fmt.Sprintf("Backup CR for %s successfully %s", instance.Name, string(op)))
		}

		// Mirror values when the data in the StatefulSet is for the current generation
		if cinderBackup.Generation == cinderBackup.Status.ObservedGeneration {
			// Mirror CinderBackup status' ReadyCount to this parent CR
			instance.Status.CinderBackupReadyCount = cinderBackup.Status.ReadyCount

			// Mirror CinderBackup's condition status
			backupCondition = cinderBackup.Status.Conditions.Mirror(cinderv1beta1.CinderBackupReadyCondition)
			instance.Status.Conditions.Set(backupCondition)
		}

	} else {
		// Clean up cinder-backup if there are no replicas
		err = r.backupCleanupDeployment(ctx, instance)
		if err != nil {
			// Should we set the condition to False?
			return ctrl.Result{}, err
		}
		// TODO: Wait for the deployment to actually disappear before setting the condition

		// The CinderBackup is ready, even if the service wasn't deployed.
		// Using "condition.DeploymentReadyMessage" here because that is what gets mirrored
		// as the message for the other Cinder children when they are successfully-deployed
		instance.Status.Conditions.MarkTrue(cinderv1beta1.CinderBackupReadyCondition, condition.DeploymentReadyMessage)
	}

	// deploy cinder-volumes
	var volumeCondition *condition.Condition
	waitingGenerationMatch := false
	for name, volume := range instance.Spec.CinderVolumes {
		cinderVolume, op, err := r.volumeDeploymentCreateOrUpdate(ctx, instance, name, volume, memcached)
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
			Log.Info(fmt.Sprintf("Volume %s CR for %s successfully %s", name, instance.Name, string(op)))
		}

		// Mirror values when the data in the StatefulSet is for the current generation
		if cinderVolume.Generation != cinderVolume.Status.ObservedGeneration {
			waitingGenerationMatch = true
		} else {
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
				c := cinderVolume.Status.Conditions.Mirror(cinderv1beta1.CinderVolumeReadyCondition)
				// Get the condition with higher priority for volumeCondition.
				volumeCondition = condition.GetHigherPrioCondition(c, volumeCondition).DeepCopy()
			}
		}
	}

	if volumeCondition != nil {
		// If there was a Status=False condition, set that as the CinderVolumeReadyCondition
		instance.Status.Conditions.Set(volumeCondition)
	} else if !waitingGenerationMatch {
		// The CinderVolumes are ready.
		// Using "condition.DeploymentReadyMessage" here because that is what gets mirrored
		// as the message for the other Cinder children when they are successfully-deployed
		instance.Status.Conditions.MarkTrue(cinderv1beta1.CinderVolumeReadyCondition, condition.DeploymentReadyMessage)
	}

	err = r.volumeCleanupDeployments(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// create CronJob
	cronjobDef := cinder.CronJob(instance, serviceLabels, serviceAnnotations)
	cronjob := cronjob.NewCronJob(
		cronjobDef,
		5*time.Second,
	)

	ctrlResult, err = cronjob.CreateOrPatch(ctx, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.CronJobReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.CronJobReadyErrorMessage,
			err.Error()))
		return ctrlResult, err
	}

	instance.Status.Conditions.MarkTrue(condition.CronJobReadyCondition, condition.CronJobReadyMessage)
	// create CronJob - end

	err = mariadbv1.DeleteUnusedMariaDBAccountFinalizers(ctx, helper, cinder.DatabaseName, instance.Spec.DatabaseAccount, instance.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	Log.Info(fmt.Sprintf("Reconciled Service '%s' successfully", instance.Name))
	// update the overall status condition if service is ready
	if instance.IsReady() {
		instance.Status.Conditions.MarkTrue(condition.ReadyCondition, condition.ReadyMessage)
	}
	return ctrl.Result{}, nil
}

// generateServiceConfigs - create Secret which hold scripts and service configuration
func (r *CinderReconciler) generateServiceConfigs(
	ctx context.Context,
	h *helper.Helper,
	instance *cinderv1beta1.Cinder,
	envVars *map[string]env.Setter,
	serviceLabels map[string]string,
	memcached *memcachedv1.Memcached,
	db *mariadbv1.Database,
) error {
	//
	// create Secret required for cinder input
	// - %-scripts holds scripts to e.g. bootstrap the service
	// - %-config holds minimal cinder config required to get the service up
	//

	labels := labels.GetLabels(instance, labels.GetGroupLabel(cinder.ServiceName), serviceLabels)

	var tlsCfg *tls.Service
	if instance.Spec.CinderAPI.TLS.Ca.CaBundleSecretName != "" {
		tlsCfg = &tls.Service{}
	}

	// customData hold any customization for all cinder services.
	customData := map[string]string{
		cinder.CustomConfigFileName: instance.Spec.CustomServiceConfig,
		cinder.MyCnfFileName:        db.GetDatabaseClientConfig(tlsCfg), //(mschuppert) for now just get the default my.cnf
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

	ospSecret, _, err := secret.GetSecret(ctx, h, instance.Spec.Secret, instance.Namespace)
	if err != nil {
		return err
	}

	transportURLSecret, _, err := secret.GetSecret(ctx, h, instance.Status.TransportURLSecret, instance.Namespace)
	if err != nil {
		return err
	}
	transportURLSecretData := string(transportURLSecret.Data["transport_url"])

	databaseAccount := db.GetAccount()
	dbSecret := db.GetSecret()

	templateParameters := make(map[string]interface{})
	templateParameters["ServiceUser"] = instance.Spec.ServiceUser
	templateParameters["ServicePassword"] = string(ospSecret.Data[instance.Spec.PasswordSelectors.Service])
	templateParameters["KeystoneInternalURL"] = keystoneInternalURL
	templateParameters["KeystonePublicURL"] = keystonePublicURL
	templateParameters["TransportURL"] = transportURLSecretData
	templateParameters["DatabaseConnection"] = fmt.Sprintf("mysql+pymysql://%s:%s@%s/%s?read_default_file=/etc/my.cnf",
		databaseAccount.Spec.UserName,
		string(dbSecret.Data[mariadbv1.DatabasePasswordSelector]),
		instance.Status.DatabaseHostname,
		cinder.DatabaseName)
	templateParameters["MemcachedServersWithInet"] = memcached.GetMemcachedServerListWithInetString()
	templateParameters["MemcachedServers"] = memcached.GetMemcachedServerListString()
	templateParameters["TimeOut"] = instance.Spec.APITimeout

	// MTLS
	if memcached.GetMemcachedMTLSSecret() != "" {
		templateParameters["MemcachedAuthCert"] = fmt.Sprint(memcachedv1.CertMountPath())
		templateParameters["MemcachedAuthKey"] = fmt.Sprint(memcachedv1.KeyMountPath())
		templateParameters["MemcachedAuthCa"] = fmt.Sprint(memcachedv1.CaMountPath())
	}

	// create httpd  vhost template parameters
	httpdVhostConfig := map[string]interface{}{}
	for _, endpt := range []service.Endpoint{service.EndpointInternal, service.EndpointPublic} {
		endptConfig := map[string]interface{}{}
		endptConfig["ServerName"] = fmt.Sprintf("%s-%s.%s.svc", cinder.ServiceName, endpt.String(), instance.Namespace)
		endptConfig["TLS"] = false // default TLS to false, and set it bellow to true if enabled
		if instance.Spec.CinderAPI.TLS.API.Enabled(endpt) {
			endptConfig["TLS"] = true
			endptConfig["SSLCertificateFile"] = fmt.Sprintf("/etc/pki/tls/certs/%s.crt", endpt.String())
			endptConfig["SSLCertificateKeyFile"] = fmt.Sprintf("/etc/pki/tls/private/%s.key", endpt.String())
		}
		httpdVhostConfig[endpt.String()] = endptConfig
	}
	templateParameters["VHosts"] = httpdVhostConfig

	var notificationInstanceURLSecret *corev1.Secret
	if instance.Status.NotificationURLSecret != nil {
		// Get a notificationInstanceURLSecret only if rabbitMQ referenced in
		// the spec is different, otherwise inherits the existing transport_url
		if instance.Spec.RabbitMqClusterName != *instance.Spec.NotificationBusInstance {
			notificationInstanceURLSecret, _, err = secret.GetSecret(ctx, h, *instance.Status.NotificationURLSecret, instance.Namespace)
			if err != nil {
				return err
			}
			templateParameters["NotificationsURL"] = string(notificationInstanceURLSecret.Data["transport_url"])
		} else {
			templateParameters["NotificationsURL"] = transportURLSecretData
		}
	}

	configTemplates := []util.Template{
		{
			Name:         fmt.Sprintf("%s-scripts", instance.Name),
			Namespace:    instance.Namespace,
			Type:         util.TemplateTypeScripts,
			InstanceType: instance.Kind,
			Labels:       labels,
		},
		{
			Name:          fmt.Sprintf("%s-config-data", instance.Name),
			Namespace:     instance.Namespace,
			Type:          util.TemplateTypeConfig,
			InstanceType:  instance.Kind,
			CustomData:    customData,
			ConfigOptions: templateParameters,
			Labels:        labels,
		},
	}

	return secret.EnsureSecrets(ctx, h, instance, configTemplates, envVars)
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
	Log := r.GetLogger(ctx)

	var hashMap map[string]string
	changed := false
	mergedMapVars := env.MergeEnvs([]corev1.EnvVar{}, envVars)
	hash, err := util.ObjectHash(mergedMapVars)
	if err != nil {
		return hash, changed, err
	}
	if hashMap, changed = util.SetHash(instance.Status.Hash, common.InputHashName, hash); changed {
		instance.Status.Hash = hashMap
		Log.Info(fmt.Sprintf("Input maps hash %s - %s", common.InputHashName, hash))
	}
	return hash, changed, nil
}

func (r *CinderReconciler) transportURLCreateOrUpdate(
	ctx context.Context,
	instance *cinderv1beta1.Cinder,
	serviceLabels map[string]string,
	rabbitMqClusterName string,
) (*rabbitmqv1.TransportURL, controllerutil.OperationResult, error) {

	// Default values used for regular messagingBus transportURL and explicitly
	// set here to ensure backward compatibility with the previous versions
	rmqName := fmt.Sprintf("%s-cinder-transport", instance.Name)
	transportURLName := instance.Spec.RabbitMqClusterName

	// When a rabbitMqClusterName is passed as input (notificationBus use case)
	// update the default rmqName and transportURLName values
	if rabbitMqClusterName != "" {
		rmqName = fmt.Sprintf("%s-cinder-transport-%s", instance.Name, rabbitMqClusterName)
		transportURLName = rabbitMqClusterName
	}
	transportURL := &rabbitmqv1.TransportURL{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rmqName,
			Namespace: instance.Namespace,
			Labels:    serviceLabels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, transportURL, func() error {
		transportURL.Spec.RabbitmqClusterName = transportURLName

		err := controllerutil.SetControllerReference(instance, transportURL, r.Scheme)
		return err
	})

	return transportURL, op, err
}

func (r *CinderReconciler) apiDeploymentCreateOrUpdate(ctx context.Context, instance *cinderv1beta1.Cinder, memcached *memcachedv1.Memcached) (*cinderv1beta1.CinderAPI, controllerutil.OperationResult, error) {
	cinderAPISpec := cinderv1beta1.CinderAPISpec{
		CinderTemplate:     instance.Spec.CinderTemplate,
		CinderAPITemplate:  instance.Spec.CinderAPI,
		ExtraMounts:        instance.Spec.ExtraMounts,
		DatabaseHostname:   instance.Status.DatabaseHostname,
		TransportURLSecret: instance.Status.TransportURLSecret,
		ServiceAccount:     instance.RbacResourceName(),
	}

	if cinderAPISpec.NodeSelector == nil {
		cinderAPISpec.NodeSelector = instance.Spec.NodeSelector
	}

	// If topology is not present in the underlying CinderAPI Spec,
	// inherit from the top-level CR
	if cinderAPISpec.TopologyRef == nil {
		cinderAPISpec.TopologyRef = instance.Spec.TopologyRef
	}

	// If memcached is not present in the underlying CinderAPI Spec,
	// inherit from the top-level CR (only when MTLS is in use)
	if memcached.GetMemcachedMTLSSecret() != "" {
		cinderAPISpec.MemcachedInstance = &instance.Spec.MemcachedInstance
	}

	deployment := &cinderv1beta1.CinderAPI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-api", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = cinderAPISpec

		if instance.Spec.NotificationBusInstance != nil {
			deployment.Spec.NotificationURLSecret = *instance.Status.NotificationURLSecret
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return deployment, op, err
}

func (r *CinderReconciler) schedulerDeploymentCreateOrUpdate(ctx context.Context, instance *cinderv1beta1.Cinder, memcached *memcachedv1.Memcached) (*cinderv1beta1.CinderScheduler, controllerutil.OperationResult, error) {
	cinderSchedulerSpec := cinderv1beta1.CinderSchedulerSpec{
		CinderTemplate:          instance.Spec.CinderTemplate,
		CinderSchedulerTemplate: instance.Spec.CinderScheduler,
		ExtraMounts:             instance.Spec.ExtraMounts,
		DatabaseHostname:        instance.Status.DatabaseHostname,
		TransportURLSecret:      instance.Status.TransportURLSecret,
		ServiceAccount:          instance.RbacResourceName(),
		TLS:                     instance.Spec.CinderAPI.TLS.Ca,
	}

	if cinderSchedulerSpec.NodeSelector == nil {
		cinderSchedulerSpec.NodeSelector = instance.Spec.NodeSelector
	}

	// If topology is not present in the underlying Scheduler Spec
	// inherit from the top-level CR
	if cinderSchedulerSpec.TopologyRef == nil {
		cinderSchedulerSpec.TopologyRef = instance.Spec.TopologyRef
	}

	// If memcached is not present in the underlying CinderScheduler Spec,
	// inherit from the top-level CR (only when MTLS is in use)
	if memcached.GetMemcachedMTLSSecret() != "" {
		cinderSchedulerSpec.MemcachedInstance = &instance.Spec.MemcachedInstance
	}

	deployment := &cinderv1beta1.CinderScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-scheduler", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = cinderSchedulerSpec

		if instance.Spec.NotificationBusInstance != nil {
			deployment.Spec.NotificationURLSecret = *instance.Status.NotificationURLSecret
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return deployment, op, err
}

func (r *CinderReconciler) backupDeploymentCreateOrUpdate(ctx context.Context, instance *cinderv1beta1.Cinder, memcached *memcachedv1.Memcached) (*cinderv1beta1.CinderBackup, controllerutil.OperationResult, error) {
	cinderBackupSpec := cinderv1beta1.CinderBackupSpec{
		CinderTemplate:       instance.Spec.CinderTemplate,
		CinderBackupTemplate: instance.Spec.CinderBackup,
		ExtraMounts:          instance.Spec.ExtraMounts,
		DatabaseHostname:     instance.Status.DatabaseHostname,
		TransportURLSecret:   instance.Status.TransportURLSecret,
		ServiceAccount:       instance.RbacResourceName(),
		TLS:                  instance.Spec.CinderAPI.TLS.Ca,
	}

	// If memcached is not present in the underlying CinderBackup Spec,
	// inherit from the top-level CR (only when MTLS is in use)
	if memcached.GetMemcachedMTLSSecret() != "" {
		cinderBackupSpec.MemcachedInstance = &instance.Spec.MemcachedInstance
	}

	if cinderBackupSpec.NodeSelector == nil {
		cinderBackupSpec.NodeSelector = instance.Spec.NodeSelector
	}

	deployment := &cinderv1beta1.CinderBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-backup", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = cinderBackupSpec

		if instance.Spec.NotificationBusInstance != nil {
			deployment.Spec.NotificationURLSecret = *instance.Status.NotificationURLSecret
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return deployment, op, err
}

func (r *CinderReconciler) backupCleanupDeployment(ctx context.Context, instance *cinderv1beta1.Cinder) error {
	deployment := &cinderv1beta1.CinderBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-backup", instance.Name),
			Namespace: instance.Namespace,
		},
	}
	key := client.ObjectKeyFromObject(deployment)
	err := r.Client.Get(ctx, key, deployment)

	if k8s_errors.IsNotFound(err) {
		// Nothing to clean up
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error looking for '%s' deployment in '%s' namespace: %w", deployment.Name, instance.Namespace, err)
	}

	err = r.Client.Delete(ctx, deployment)
	if err != nil && !k8s_errors.IsNotFound(err) {
		return fmt.Errorf("Error cleaning up %s: %w", deployment.Name, err)
	}

	return nil
}

func (r *CinderReconciler) volumeDeploymentCreateOrUpdate(ctx context.Context, instance *cinderv1beta1.Cinder, name string, volTemplate cinderv1beta1.CinderVolumeTemplate, memcached *memcachedv1.Memcached) (*cinderv1beta1.CinderVolume, controllerutil.OperationResult, error) {
	cinderVolumeSpec := cinderv1beta1.CinderVolumeSpec{
		CinderTemplate:       instance.Spec.CinderTemplate,
		CinderVolumeTemplate: volTemplate,
		ExtraMounts:          instance.Spec.ExtraMounts,
		DatabaseHostname:     instance.Status.DatabaseHostname,
		TransportURLSecret:   instance.Status.TransportURLSecret,
		ServiceAccount:       instance.RbacResourceName(),
		TLS:                  instance.Spec.CinderAPI.TLS.Ca,
	}

	if cinderVolumeSpec.CinderVolumeTemplate.NodeSelector == nil {
		cinderVolumeSpec.CinderVolumeTemplate.NodeSelector = instance.Spec.NodeSelector
	}

	if cinderVolumeSpec.CinderVolumeTemplate.TopologyRef == nil {
		cinderVolumeSpec.CinderVolumeTemplate.TopologyRef = instance.Spec.TopologyRef
	}

	// If memcached is not present in the underlying CinderVolume Spec,
	// inherit from the top-level CR (only when MTLS is in use)
	if memcached.GetMemcachedMTLSSecret() != "" {
		cinderVolumeSpec.MemcachedInstance = &instance.Spec.MemcachedInstance
	}

	deployment := &cinderv1beta1.CinderVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-volume-%s", instance.Name, name),
			Namespace: instance.Namespace,
			Labels:    map[string]string{cinderv1beta1.Backend: name},
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = cinderVolumeSpec

		if instance.Spec.NotificationBusInstance != nil {
			deployment.Spec.NotificationURLSecret = *instance.Status.NotificationURLSecret
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
	Log := r.GetLogger(ctx)

	// Generate a list of volume CRs
	volumes := &cinderv1beta1.CinderVolumeList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.Namespace),
	}
	if err := r.Client.List(ctx, volumes, listOpts...); err != nil {
		Log.Error(err, "Unable to retrieve volume CRs %v")
		return nil
	}

	for _, volume := range volumes.Items {
		// Skip volumes that we don't own
		if cinder.GetOwningCinderName(&volume) != instance.Name {
			continue
		}

		// Delete the volume if it's no longer in the spec
		_, exists := instance.Spec.CinderVolumes[volume.BackendName()]
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

func (r *CinderReconciler) ensureDB(
	ctx context.Context,
	h *helper.Helper,
	instance *cinderv1beta1.Cinder,
) (*mariadbv1.Database, ctrl.Result, error) {
	Log := r.GetLogger(ctx)

	// ensure MariaDBAccount exists.  This account record may be created by
	// openstack-operator or the cloud operator up front without a specific
	// MariaDBDatabase configured yet.   Otherwise, a MariaDBAccount CR is
	// created here with a generated username as well as a secret with
	// generated password.   The MariaDBAccount is created without being
	// yet associated with any MariaDBDatabase.
	_, _, err := mariadbv1.EnsureMariaDBAccount(
		ctx, h, instance.Spec.DatabaseAccount,
		instance.Namespace, false, "cinder",
	)

	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			mariadbv1.MariaDBAccountReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			mariadbv1.MariaDBAccountNotReadyMessage,
			err.Error()))

		return nil, ctrl.Result{}, err
	}
	instance.Status.Conditions.MarkTrue(
		mariadbv1.MariaDBAccountReadyCondition,
		mariadbv1.MariaDBAccountReadyMessage,
	)

	db := mariadbv1.NewDatabaseForAccount(
		instance.Spec.DatabaseInstance, // mariadb/galera service to target
		cinder.DatabaseName,            // name used in CREATE DATABASE in mariadb
		cinder.DatabaseName,            // CR name for MariaDBDatabase
		instance.Spec.DatabaseAccount,  // CR name for MariaDBAccount
		instance.Namespace,             // namespace
	)

	// create or patch the DB
	ctrlResult, err := db.CreateOrPatchAll(ctx, h)

	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBReadyErrorMessage,
			err.Error()))
		return db, ctrl.Result{}, err
	}
	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBReadyRunningMessage))
		return db, ctrlResult, nil
	}
	// wait for the DB to be setup
	// (ksambor) should we use WaitForDBCreatedWithTimeout instead?
	ctrlResult, err = db.WaitForDBCreated(ctx, h)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBReadyErrorMessage,
			err.Error()))
		return db, ctrlResult, err
	}
	if (ctrlResult != ctrl.Result{}) {
		Log.Info(fmt.Sprintf("%s... requeueing", condition.DBReadyRunningMessage))
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBReadyRunningMessage))
		return db, ctrlResult, nil
	}

	// update Status.DatabaseHostname, used to config the service
	instance.Status.DatabaseHostname = db.GetDatabaseHostname()
	instance.Status.Conditions.MarkTrue(condition.DBReadyCondition, condition.DBReadyMessage)
	return db, ctrlResult, nil
}
