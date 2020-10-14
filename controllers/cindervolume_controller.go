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
	"github.com/prometheus/common/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	cindervolume "github.com/openstack-k8s-operators/cinder-operator/pkg/cindervolume"
	common "github.com/openstack-k8s-operators/cinder-operator/pkg/common"
	util "github.com/openstack-k8s-operators/lib-common/pkg/util"
)

// CinderVolumeReconciler reconciles a CinderVolume object
type CinderVolumeReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

// GetClient -
func (r *CinderVolumeReconciler) GetClient() client.Client {
	return r.Client
}

// GetLogger -
func (r *CinderVolumeReconciler) GetLogger() logr.Logger {
	return r.Log
}

// GetScheme -
func (r *CinderVolumeReconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;create;update;delete;
// +kubebuilder:rbac:groups=security.openshift.io,namespace=openstack,resources=securitycontextconstraints,resourceNames=privileged,verbs=use

// Reconcile - cinder volume
func (r *CinderVolumeReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("cindervolume", req.NamespacedName)

	// Fetch the CinderVolume instance
	instance := &cinderv1beta1.CinderVolume{}
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
	hashes := []cinderv1beta1.Hash{}
	secretHashes, err := common.GetSecretsFromCR(r, instance, instance.Namespace, instance.Spec, &envVars)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}
	hashes = append(hashes, secretHashes...)

	// check for required configMaps
	configMaps := []string{
		fmt.Sprintf("%s-scripts", instance.Spec.ManagingCrName),            //ScriptsConfigMap
		fmt.Sprintf("%s-config-data", instance.Spec.ManagingCrName),        //ConfigMap
		fmt.Sprintf("%s-config-data-custom", instance.Spec.ManagingCrName), //CustomConfigMap
		fmt.Sprintf("%s-config-data-custom", instance.Name),                //CustomConfigMap per volume service

	}

	configHashes, err := common.GetConfigMaps(r, instance, configMaps, instance.Namespace, &envVars, instance.Spec.ManagingCrName)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}
	hashes = append(hashes, configHashes...)

	// update Hashes in CR status
	err = common.UpdateStatusHash(r, instance, &instance.Status.Hashes, hashes)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}

	// cinder-volume
	// Create or update the Deployment object
	op, err := r.statefulsetCreateOrUpdate(instance, envVars)
	if err != nil {
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		r.Log.Info(fmt.Sprintf("StatefulSet %s successfully reconciled - operation: %s", instance.Name, string(op)))
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager -
func (r *CinderVolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// watch for configmap where the CM upper-cr label AND the CR.Spec.ManagingCrName label matches
	configMapFn := handler.ToRequestsFunc(func(cm handler.MapObject) []reconcile.Request {
		result := []reconcile.Request{}

		// get all volume CRs
		volumes := &cinderv1beta1.CinderVolumeList{}
		listOpts := []client.ListOption{
			client.InNamespace(cm.Meta.GetNamespace()),
		}
		if err := r.Client.List(context.Background(), volumes, listOpts...); err != nil {
			log.Error(err, "Unable to retrieve volume CRs %v")
			return nil
		}

		label := cm.Meta.GetLabels()
		// verify object has upper-cr label
		if l, ok := label["upper-cr"]; ok {
			for _, cr := range volumes.Items {
				// return reconcil event for the CR where the CM upper-cr label AND the CR.Spec.ManagingCrName label matches
				if l == cr.Spec.ManagingCrName {
					// return namespace and Name of CR
					name := client.ObjectKey{
						Namespace: cm.Meta.GetNamespace(),
						Name:      cr.Name,
					}
					r.Log.Info(fmt.Sprintf("ConfigMap object %s and CR %s marked with label: %s", cm.Meta.GetName(), cr.Name, l))

					result = append(result, reconcile.Request{NamespacedName: name})
				}
			}
		}
		if len(result) > 0 {
			return result
		}
		return nil
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&cinderv1beta1.CinderVolume{}).
		Owns(&appsv1.StatefulSet{}).
		// watch the config CMs we don't own
		Watches(&source.Kind{Type: &corev1.ConfigMap{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: configMapFn,
			}).
		Complete(r)
}

func (r *CinderVolumeReconciler) statefulsetCreateOrUpdate(instance *cinderv1beta1.CinderVolume, envVars map[string]util.EnvSetter) (controllerutil.OperationResult, error) {
	trueVar := true

	// set KOLLA_CONFIG env vars
	envVars["KOLLA_CONFIG_FILE"] = util.EnvValue(cindervolume.KollaConfig)
	envVars["KOLLA_CONFIG_STRATEGY"] = util.EnvValue("COPY_ALWAYS")

	// get readinessProbes
	readinessProbe := util.Probe{ProbeType: "readiness"}
	livenessProbe := util.Probe{ProbeType: "liveness"}

	// get volumes
	initVolumeMounts := common.GetInitVolumeMounts()
	// add cinder-volume specific InitVolumeMounts
	for _, volMount := range cindervolume.GetInitVolumeMounts() {
		initVolumeMounts = append(initVolumeMounts, volMount)
	}

	volumeMounts := common.GetVolumeMounts()
	// add cinder-volume specific VolumeMounts
	for _, volMount := range cindervolume.GetVolumeMounts() {
		volumeMounts = append(volumeMounts, volMount)
	}

	volumes := common.GetVolumes(instance.Spec.ManagingCrName)
	// add cinder-volume Volumes
	for _, vol := range cindervolume.GetVolumes(instance.Name) {
		volumes = append(volumes, vol)
	}

	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, statefulset, func() error {

		// statefulset selector is immutable so we set this value only if
		// a new object is going to be created
		if statefulset.ObjectMeta.CreationTimestamp.IsZero() {
			statefulset.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: common.GetLabels(instance.Name, cindervolume.AppLabel),
			}
		}

		if len(statefulset.Spec.Template.Spec.Containers) != 1 {
			statefulset.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
		}
		envs := util.MergeEnvs(statefulset.Spec.Template.Spec.Containers[0].Env, envVars)

		// labels
		common.InitLabelMap(&statefulset.Spec.Template.Labels)
		for k, v := range common.GetLabels(instance.Name, cindervolume.AppLabel) {
			statefulset.Spec.Template.Labels[k] = v
		}

		statefulset.Spec.Replicas = &instance.Spec.Replicas
		statefulset.Spec.Template.Spec = corev1.PodSpec{
			ServiceAccountName: serviceAccountName,
			NodeSelector:       common.GetNodeSelector(instance.Spec.NodeSelectorRoleName),
			HostNetwork:        true,
			HostIPC:            true,
			DNSPolicy:          "ClusterFirstWithHostNet",
			Volumes:            volumes,
			Containers: []corev1.Container{
				{
					Name:  "cinder-volume",
					Image: instance.Spec.ContainerImage,
					SecurityContext: &corev1.SecurityContext{
						Privileged: &trueVar,
					},
					ReadinessProbe: readinessProbe.GetProbe(),
					LivenessProbe:  livenessProbe.GetProbe(),
					Env:            envs,
					VolumeMounts:   volumeMounts,
				},
			},
		}

		initContainerDetails := common.InitContainer{
			Privileged:     true,
			ContainerImage: instance.Spec.ContainerImage,
			DatabaseHost:   instance.Spec.DatabaseHostname,
			CinderSecret:   instance.Spec.CinderSecret,
			NovaSecret:     instance.Spec.NovaSecret,
			VolumeMounts:   initVolumeMounts,
		}

		statefulset.Spec.Template.Spec.InitContainers = common.GetInitContainer(initContainerDetails)

		err := controllerutil.SetControllerReference(instance, statefulset, r.Scheme)
		if err != nil {
			return err
		}

		return nil
	})

	return op, err
}
