/*


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

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	cinder "github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	common "github.com/openstack-k8s-operators/cinder-operator/pkg/common"
	keystonev1beta1 "github.com/openstack-k8s-operators/keystone-operator/api/v1beta1"
	util "github.com/openstack-k8s-operators/lib-common/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

// CinderReconciler reconciles a Cinder object
type CinderReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

// GetClient -
func (r *CinderReconciler) GetClient() client.Client {
	return r.Client
}

// GetLogger -
func (r *CinderReconciler) GetLogger() logr.Logger {
	return r.Log
}

// GetScheme -
func (r *CinderReconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderapis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderapis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderschedulers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderschedulers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cinderbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;create;update;delete;

// Reconcile - cinder
func (r *CinderReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("cinder", req.NamespacedName)

	// Fetch the Cinder instance
	instance := &cinderv1beta1.Cinder{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
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

	envVars := make(map[string]util.EnvSetter)

	// check for required secrets
	cinderSecret, hash, err := common.GetSecret(r.Client, instance.Spec.CinderSecret, instance.Namespace)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}
	envVars[instance.Spec.CinderSecret] = util.EnvValue(hash)

	_, hash, err = common.GetSecret(r.Client, instance.Spec.NovaSecret, instance.Namespace)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}
	envVars[instance.Spec.NovaSecret] = util.EnvValue(hash)

	// Create/update configmaps from templates
	cmLabels := common.GetLabels(instance.Name, cinder.AppLabel)
	cmLabels["upper-cr"] = instance.Name

	cms := []common.ConfigMap{
		// ScriptsConfigMap
		{
			Name:           fmt.Sprintf("%s-scripts", instance.Name),
			Namespace:      instance.Namespace,
			CMType:         common.CMTypeScripts,
			InstanceType:   instance.Kind,
			AdditionalData: map[string]string{},
			Labels:         cmLabels,
		},
		// ConfigMap
		{
			Name:           fmt.Sprintf("%s-config-data", instance.Name),
			Namespace:      instance.Namespace,
			CMType:         common.CMTypeConfig,
			InstanceType:   instance.Kind,
			AdditionalData: map[string]string{},
			Labels:         cmLabels,
		},
		// CustomConfigMap
		{
			Name:      fmt.Sprintf("%s-config-data-custom", instance.Name),
			Namespace: instance.Namespace,
			CMType:    common.CMTypeCustom,
			Labels:    cmLabels,
		},
	}
	err = common.EnsureConfigMaps(r, instance, cms, &envVars)
	if err != nil {
		return ctrl.Result{}, nil
	}

	// Create cinder DB
	db := common.Database{
		DatabaseName:     dbName,
		DatabaseHostname: instance.Spec.DatabaseHostname,
		Secret:           instance.Spec.CinderSecret,
	}
	databaseObj, err := common.DatabaseObject(r, instance, db)
	if err != nil {
		return ctrl.Result{}, err
	}
	// set owner reference on databaseObj
	oref := metav1.NewControllerRef(instance, instance.GroupVersionKind())
	databaseObj.SetOwnerReferences([]metav1.OwnerReference{*oref})

	foundDatabase := &unstructured.Unstructured{}
	foundDatabase.SetGroupVersionKind(databaseObj.GroupVersionKind())
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: databaseObj.GetName(), Namespace: databaseObj.GetNamespace()}, foundDatabase)
	if err != nil && k8s_errors.IsNotFound(err) {
		err := r.Client.Create(context.TODO(), &databaseObj)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		completed, _, err := unstructured.NestedBool(foundDatabase.UnstructuredContent(), "status", "completed")
		if !completed {
			r.Log.Info(fmt.Sprintf("Waiting on %s DB to be created...", dbName))
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}
	}

	// run dbsync job
	job := cinder.DbSyncJob(instance, r.Scheme)
	dbSyncHash, err := util.ObjectHash(job)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error calculating DB sync hash: %v", err)
	}

	requeue := true
	if instance.Status.DbSyncHash != dbSyncHash {
		requeue, err = util.EnsureJob(job, r.Client, r.Log)
		r.Log.Info("Running DB sync")
		if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			r.Log.Info("Waiting on DB sync")
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}
	}
	// db sync completed... okay to store the hash to disable it
	if err := r.setDbSyncHash(instance, dbSyncHash); err != nil {
		return ctrl.Result{}, err
	}

	// delete the dbsync job
	requeue, err = util.DeleteJob(job, r.Kclient, r.Log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// deploy cinder-api
	// Create or update the nova-api Deployment object
	op, err := r.apiDeploymentCreateOrUpdate(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		r.Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// cinder service
	selector := make(map[string]string)
	selector["app"] = fmt.Sprintf("%s-api", instance.Name)

	serviceInfo := common.ServiceDetails{
		Name:      instance.Name,
		Namespace: instance.Namespace,
		AppLabel:  "cinder-api",
		Selector:  selector,
		Port:      8776,
	}

	service := &corev1.Service{}
	service.Name = serviceInfo.Name
	service.Namespace = serviceInfo.Namespace
	if err := controllerutil.SetControllerReference(instance, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	service, op, err = common.CreateOrUpdateService(r.Client, r.Log, service, &serviceInfo)
	if err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("Service successfully reconciled", "operation", op)

	// Create the route if none exists
	routeInfo := common.RouteDetails{
		Name:      instance.Name,
		Namespace: instance.Namespace,
		AppLabel:  "cinder-api",
		Port:      "api",
	}
	route := common.Route(routeInfo)
	if err := controllerutil.SetControllerReference(instance, route, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	err = common.CreateOrUpdateRoute(r.Client, r.Log, route)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update status with endpoint information
	// Keystone setup
	cinderv2KeystoneService := &keystonev1beta1.KeystoneService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cinderv2",
			Namespace: instance.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, cinderv2KeystoneService, func() error {
		cinderv2KeystoneService.Spec.Username = "cinder"
		cinderv2KeystoneService.Spec.Password = string(cinderSecret.Data["CinderKeystoneAuthPassword"])
		cinderv2KeystoneService.Spec.ServiceType = "volumev2"
		cinderv2KeystoneService.Spec.ServiceName = "cinderv2"
		cinderv2KeystoneService.Spec.ServiceDescription = "OpenStack Block Storage"
		cinderv2KeystoneService.Spec.Enabled = true
		cinderv2KeystoneService.Spec.Region = "regionOne"
		cinderv2KeystoneService.Spec.AdminURL = fmt.Sprintf("http://%s/v2/%%(project_id)s", route.Spec.Host)
		cinderv2KeystoneService.Spec.PublicURL = fmt.Sprintf("http://%s/v2/%%(project_id)s", route.Spec.Host)
		cinderv2KeystoneService.Spec.InternalURL = fmt.Sprintf("http://%s.openstack.svc:8776/v2/%%(project_id)s", instance.Name)
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	cinderv3KeystoneService := &keystonev1beta1.KeystoneService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cinderv3",
			Namespace: instance.Namespace,
		},
	}

	// don't pass the cinder user again as update is right now not handled in KeystoneService
	// atm we'll get a cinderv3 user which is not used.
	_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, cinderv3KeystoneService, func() error {
		cinderv3KeystoneService.Spec.ServiceType = "volumev3"
		cinderv3KeystoneService.Spec.ServiceName = "cinderv3"
		cinderv3KeystoneService.Spec.ServiceDescription = "OpenStack Block Storage"
		cinderv3KeystoneService.Spec.Enabled = true
		cinderv3KeystoneService.Spec.Region = "regionOne"
		cinderv3KeystoneService.Spec.AdminURL = fmt.Sprintf("http://%s/v3/%%(project_id)s", route.Spec.Host)
		cinderv3KeystoneService.Spec.PublicURL = fmt.Sprintf("http://%s/v3/%%(project_id)s", route.Spec.Host)
		cinderv3KeystoneService.Spec.InternalURL = fmt.Sprintf("http://%s.openstack.svc:8776/v3/%%(project_id)s", instance.Name)
		return nil
	})

	if err != nil {
		return ctrl.Result{}, err
	}

	r.setAPIEndpoint(instance, cinderv2KeystoneService.Spec.PublicURL)

	// deploy cinder-scheduler
	// Create or update the cinder-scheduler Deployment object
	op, err = r.schedulerDeploymentCreateOrUpdate(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		r.Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// deploy cinder-backup
	op, err = r.backupDeploymentCreateOrUpdate(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		r.Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// deploy cinder-volume services
	// calc Spec.CinderVolumes hash to verify if any of the volume services changed
	cinderVolumesHash, err := util.ObjectHash(instance.Spec.CinderVolumes)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error volumes hash: %v", err)
	}
	r.Log.Info("VolumesHash: ", "Cinder-Volumes Hash:", cinderVolumesHash)

	for _, volume := range instance.Spec.CinderVolumes {
		// VolumeCustomConfigMap
		volumeCMName := fmt.Sprintf("%s-%s-config-data-custom", instance.Name, volume.Name)
		cm := []common.ConfigMap{
			// CustomConfigMap
			{
				Name:      volumeCMName,
				Namespace: instance.Namespace,
				CMType:    common.CMTypeCustom,
				Labels:    cmLabels,
			},
		}
		err = common.EnsureConfigMaps(r, instance, cm, &envVars)
		if err != nil {
			return ctrl.Result{}, nil
		}

		op, err = r.volumeDeploymentCreateOrUpdate(instance, &volume)
		if err != nil {
			return ctrl.Result{}, err
		}
		if op != controllerutil.OperationResultNone {
			r.Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", volume.Name, string(op)))
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager -
func (r *CinderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cinderv1beta1.Cinder{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&cinderv1beta1.CinderAPI{}).
		Owns(&cinderv1beta1.CinderScheduler{}).
		Owns(&cinderv1beta1.CinderBackup{}).
		Owns(&cinderv1beta1.CinderVolume{}).
		Owns(&routev1.Route{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *CinderReconciler) setDbSyncHash(api *cinderv1beta1.Cinder, hashStr string) error {

	if hashStr != api.Status.DbSyncHash {
		api.Status.DbSyncHash = hashStr
		if err := r.Client.Status().Update(context.TODO(), api); err != nil {
			return err
		}
	}
	return nil
}

func (r *CinderReconciler) setAPIEndpoint(instance *cinderv1beta1.Cinder, endpoint string) error {

	if endpoint != instance.Status.APIEndpoint {
		instance.Status.APIEndpoint = endpoint
		if err := r.Client.Status().Update(context.TODO(), instance); err != nil {
			return err
		}
	}
	return nil

}

func (r *CinderReconciler) apiDeploymentCreateOrUpdate(instance *cinderv1beta1.Cinder) (controllerutil.OperationResult, error) {
	deployment := &cinderv1beta1.CinderAPI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-api", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, deployment, func() error {
		deployment.Spec = cinderv1beta1.CinderAPISpec{
			ManagingCrName:   instance.Name,
			DatabaseHostname: instance.Spec.DatabaseHostname,
			NovaSecret:       instance.Spec.NovaSecret,
			CinderSecret:     instance.Spec.CinderSecret,
			Replicas:         instance.Spec.CinderAPIReplicas,
			ContainerImage:   instance.Spec.CinderAPIContainerImage,
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return op, err
}

func (r *CinderReconciler) schedulerDeploymentCreateOrUpdate(instance *cinderv1beta1.Cinder) (controllerutil.OperationResult, error) {
	deployment := &cinderv1beta1.CinderScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-scheduler", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, deployment, func() error {
		deployment.Spec = cinderv1beta1.CinderSchedulerSpec{
			ManagingCrName:   instance.Name,
			DatabaseHostname: instance.Spec.DatabaseHostname,
			NovaSecret:       instance.Spec.NovaSecret,
			CinderSecret:     instance.Spec.CinderSecret,
			Replicas:         instance.Spec.CinderSchedulerReplicas,
			ContainerImage:   instance.Spec.CinderSchedulerContainerImage,
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return op, err
}

func (r *CinderReconciler) backupDeploymentCreateOrUpdate(instance *cinderv1beta1.Cinder) (controllerutil.OperationResult, error) {
	deployment := &cinderv1beta1.CinderBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-backup", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, deployment, func() error {
		deployment.Spec = cinderv1beta1.CinderBackupSpec{
			ManagingCrName:       instance.Name,
			NodeSelectorRoleName: instance.Spec.CinderBackupNodeSelectorRoleName,
			DatabaseHostname:     instance.Spec.DatabaseHostname,
			NovaSecret:           instance.Spec.NovaSecret,
			CinderSecret:         instance.Spec.CinderSecret,
			Replicas:             instance.Spec.CinderBackupReplicas,
			ContainerImage:       instance.Spec.CinderBackupContainerImage,
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return op, err
}

func (r *CinderReconciler) volumeDeploymentCreateOrUpdate(instance *cinderv1beta1.Cinder, volume *cinderv1beta1.Volume) (controllerutil.OperationResult, error) {
	deployment := &cinderv1beta1.CinderVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", instance.Name, volume.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, deployment, func() error {
		deployment.Spec = cinderv1beta1.CinderVolumeSpec{
			ManagingCrName:       instance.Name,
			DatabaseHostname:     instance.Spec.DatabaseHostname,
			CinderSecret:         instance.Spec.CinderSecret,
			NovaSecret:           instance.Spec.NovaSecret,
			ContainerImage:       volume.CinderVolumeContainerImage,
			NodeSelectorRoleName: volume.CinderVolumeNodeSelectorRoleName,
			Replicas:             volume.CinderVolumeReplicas,
		}

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return op, err
}
