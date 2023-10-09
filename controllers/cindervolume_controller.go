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
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	"github.com/openstack-k8s-operators/cinder-operator/pkg/cindervolume"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/statefulset"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
)

// GetClient -
func (r *CinderVolumeReconciler) GetClient() client.Client {
	return r.Client
}

// GetKClient -
func (r *CinderVolumeReconciler) GetKClient() kubernetes.Interface {
	return r.Kclient
}

// GetLogger -
func (r *CinderVolumeReconciler) GetLogger() logr.Logger {
	return r.Log
}

// GetScheme -
func (r *CinderVolumeReconciler) GetScheme() *runtime.Scheme {
	return r.Scheme
}

// CinderVolumeReconciler reconciles a Cinder object
type CinderVolumeReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

//+kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cinder.openstack.org,resources=cindervolumes/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=security.openshift.io,namespace=openstack,resources=securitycontextconstraints,resourceNames=privileged,verbs=use
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

// Reconcile -
func (r *CinderVolumeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	_ = log.FromContext(ctx)

	// Fetch the CinderVolume instance
	instance := &cinderv1beta1.CinderVolume{}
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
			condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
			condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
			condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
			condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
		)

		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback e.g. in the cli
		return ctrl.Result{}, nil
	}
	if instance.Status.Hash == nil {
		instance.Status.Hash = map[string]string{}
	}
	if instance.Status.NetworkAttachments == nil {
		instance.Status.NetworkAttachments = map[string][]string{}
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, instance, helper)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CinderVolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Watch for changes to secrets we don't own. Global secrets
	// (e.g. TransportURLSecret) are handled by the main cinder controller.
	secretFn := func(o client.Object) []reconcile.Request {
		var namespace string = o.GetNamespace()
		var secretName string = o.GetName()
		result := []reconcile.Request{}

		// get all volume CRs
		volumes := &cinderv1beta1.CinderVolumeList{}
		listOpts := []client.ListOption{
			client.InNamespace(namespace),
		}
		if err := r.Client.List(context.Background(), volumes, listOpts...); err != nil {
			r.Log.Error(err, "Unable to retrieve volume CRs %v")
			return nil
		}

		// Watch for changes to secrets where the owner label AND the
		// CR.Spec.ManagingCrName label matches
		label := o.GetLabels()
		if l, ok := label[labels.GetOwnerNameLabelSelector(labels.GetGroupLabel(cinder.ServiceName))]; ok {
			for _, cr := range volumes.Items {
				// return reconcile event for the CR where the owner label AND the parentCinderName matches
				if l == cinder.GetOwningCinderName(&cr) {
					// return namespace and Name of CR
					name := client.ObjectKey{
						Namespace: o.GetNamespace(),
						Name:      cr.Name,
					}
					r.Log.Info(fmt.Sprintf("Secret %s and CR %s marked with label: %s", o.GetName(), cr.Name, l))

					result = append(result, reconcile.Request{NamespacedName: name})
				}
			}
		}

		// Watch for changes to any CustomServiceConfigSecrets
		for _, cr := range volumes.Items {
			for _, v := range cr.Spec.CustomServiceConfigSecrets {
				if v == secretName {
					name := client.ObjectKey{
						Namespace: namespace,
						Name:      cr.Name,
					}
					r.Log.Info(fmt.Sprintf("Secret %s is used by Cinder CR %s", secretName, cr.Name))
					result = append(result, reconcile.Request{NamespacedName: name})
				}
			}
		}
		if len(result) > 0 {
			return result
		}
		return nil
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cinderv1beta1.CinderVolume{}).
		Owns(&appsv1.StatefulSet{}).
		// watch the secrets we don't own
		Watches(&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(secretFn)).
		Complete(r)
}

func (r *CinderVolumeReconciler) reconcileDelete(ctx context.Context, instance *cinderv1beta1.CinderVolume, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s' delete", instance.Name))

	// Service is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' delete successfully", instance.Name))

	return ctrl.Result{}, nil
}

func (r *CinderVolumeReconciler) reconcileInit(
	ctx context.Context,
	instance *cinderv1beta1.CinderVolume,
	helper *helper.Helper,
) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s' init", instance.Name))

	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' init successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *CinderVolumeReconciler) reconcileNormal(ctx context.Context, instance *cinderv1beta1.CinderVolume, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s'", instance.Name))

	configVars := make(map[string]env.Setter)

	//
	// check for required OpenStack secret holding passwords for service/admin user and add hash to the vars map
	//
	ctrlResult, err := r.getSecret(ctx, helper, instance, instance.Spec.Secret, &configVars)
	if err != nil {
		return ctrlResult, err
	}

	//
	// check for required TransportURL secret holding transport URL string
	//
	ctrlResult, err = r.getSecret(ctx, helper, instance, instance.Spec.TransportURLSecret, &configVars)
	if err != nil {
		return ctrlResult, err
	}

	//
	// check for required service secrets
	//
	for _, secretName := range instance.Spec.CustomServiceConfigSecrets {
		ctrlResult, err = r.getSecret(ctx, helper, instance, secretName, &configVars)
		if err != nil {
			return ctrlResult, err
		}
	}

	//
	// check for required Cinder secrets that should have been created by parent Cinder CR
	//
	parentCinderName := cinder.GetOwningCinderName(instance)
	parentSecrets := []string{
		fmt.Sprintf("%s-scripts", parentCinderName),
		fmt.Sprintf("%s-config-data", parentCinderName),
	}

	for _, parentSecret := range parentSecrets {
		ctrlResult, err = r.getSecret(ctx, helper, instance, parentSecret, &configVars)
		if err != nil {
			return ctrlResult, err
		}
	}
	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)

	//
	// Create secrets required as input for the Service and calculate an overall hash of hashes
	//
	serviceLabels := map[string]string{
		common.AppSelector:       cinder.ServiceName,
		common.ComponentSelector: cindervolume.Component,
		cindervolume.Backend:     instance.Name[len(cindervolume.Component)+1:],
	}

	//
	// create custom Configmap for this cinder volume service
	//
	err = r.generateServiceConfigs(ctx, helper, instance, &configVars, serviceLabels)
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
	inputHash, hashChanged, err := r.createHashOfInputHashes(ctx, instance, configVars)
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
	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)

	//
	// TODO check when/if Init, Update, or Upgrade should/could be skipped
	//

	// networks to attach to
	for _, netAtt := range instance.Spec.NetworkAttachments {
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

	serviceAnnotations, err := nad.CreateNetworksAnnotation(instance.Namespace, instance.Spec.NetworkAttachments)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.NetworkAttachments, err)
	}

	// Handle service init
	ctrlResult, err = r.reconcileInit(ctx, instance, helper)
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

	// Deploy a statefulset
	ssDef := cindervolume.StatefulSet(instance, inputHash, serviceLabels, serviceAnnotations)
	ss := statefulset.NewStatefulSet(
		ssDef,
		time.Duration(5)*time.Second,
	)

	ctrlResult, err = ss.CreateOrPatch(ctx, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DeploymentReadyErrorMessage,
			err.Error()))
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))
		return ctrlResult, nil
	}
	instance.Status.ReadyCount = ss.GetStatefulSet().Status.ReadyReplicas

	// verify if network attachment matches expectations
	networkReady := false
	networkAttachmentStatus := map[string][]string{}
	if *instance.Spec.Replicas > 0 {
		networkReady, networkAttachmentStatus, err = nad.VerifyNetworkStatusFromAnnotation(
			ctx,
			helper,
			instance.Spec.NetworkAttachments,
			serviceLabels,
			instance.Status.ReadyCount,
		)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		networkReady = true
	}

	instance.Status.NetworkAttachments = networkAttachmentStatus
	if networkReady {
		instance.Status.Conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)
	} else {
		err := fmt.Errorf("not all pods have interfaces with ips as configured in NetworkAttachments: %s", instance.Spec.NetworkAttachments)
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.NetworkAttachmentsReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.NetworkAttachmentsReadyErrorMessage,
			err.Error()))

		return ctrl.Result{}, err
	}

	if instance.Status.ReadyCount > 0 {
		instance.Status.Conditions.MarkTrue(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)
	}
	// create StatefulSet - end

	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *CinderVolumeReconciler) reconcileUpdate(ctx context.Context, instance *cinderv1beta1.CinderVolume, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s' update", instance.Name))

	// TODO: should have minor update tasks if required
	// - delete dbsync hash from status to rerun it?

	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' update successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *CinderVolumeReconciler) reconcileUpgrade(ctx context.Context, instance *cinderv1beta1.CinderVolume, helper *helper.Helper) (ctrl.Result, error) {
	r.Log.Info(fmt.Sprintf("Reconciling Service '%s' upgrade", instance.Name))

	// TODO: should have major version upgrade tasks
	// -delete dbsync hash from status to rerun it?

	r.Log.Info(fmt.Sprintf("Reconciled Service '%s' upgrade successfully", instance.Name))
	return ctrl.Result{}, nil
}

// getSecret - get the specified secret, and add its hash to envVars
func (r *CinderVolumeReconciler) getSecret(
	ctx context.Context,
	h *helper.Helper,
	instance *cinderv1beta1.CinderVolume,
	secretName string,
	envVars *map[string]env.Setter,
) (ctrl.Result, error) {
	secret, hash, err := secret.GetSecret(ctx, h, secretName, instance.Namespace)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.InputReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				condition.InputReadyWaitingMessage))
			return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, fmt.Errorf("Secret %s not found", secretName)
		}
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.InputReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.InputReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	// Add a prefix to the var name to avoid accidental collision with other non-secret
	// vars. The secret names themselves will be unique.
	(*envVars)["secret-"+secret.Name] = env.SetValue(hash)

	return ctrl.Result{}, nil
}

// generateServiceConfigs - create Secret which holds the service configuration
// TODO add DefaultConfigOverwrite
func (r *CinderVolumeReconciler) generateServiceConfigs(
	ctx context.Context,
	h *helper.Helper,
	instance *cinderv1beta1.CinderVolume,
	envVars *map[string]env.Setter,
	serviceLabels map[string]string,
) error {
	//
	// create custom Secret for cinder service-specific config input
	// - %-config-data holds custom config for the service
	//

	labels := labels.GetLabels(instance, labels.GetGroupLabel(cinder.ServiceName), serviceLabels)

	// customData hold any customization for the service.
	customData := map[string]string{}

	for key, data := range instance.Spec.DefaultConfigOverwrite {
		customData[key] = data
	}

	// Fetch the two service config snippets (DefaultsConfigFileName and
	// CustomConfigFileName) from the Secret generated by the top level
	// cinder controller, and add them to this service specific Secret.
	cinderSecretName := cinder.GetOwningCinderName(instance) + "-config-data"
	cinderSecret, _, err := secret.GetSecret(ctx, h, cinderSecretName, instance.Namespace)
	if err != nil {
		return err
	}
	customData[cinder.DefaultsConfigFileName] = string(cinderSecret.Data[cinder.DefaultsConfigFileName])
	customData[cinder.CustomConfigFileName] = string(cinderSecret.Data[cinder.CustomConfigFileName])

	customSecrets := ""
	for _, secretName := range instance.Spec.CustomServiceConfigSecrets {
		secret, _, err := secret.GetSecret(ctx, h, secretName, instance.Namespace)
		if err != nil {
			return err
		}
		for _, data := range secret.Data {
			customSecrets += string(data) + "\n"
		}
	}
	customData[cinder.CustomServiceConfigSecretsFileName] = customSecrets

	usesLVM, customServiceConfig := processCustomServiceConfig(instance.Spec.CustomServiceConfig)
	customData[cinder.CustomServiceConfigFileName] = customServiceConfig

	templateParameters := make(map[string]interface{})
	if usesLVM {
		networkAttachmentAddrs := cinder.GetNetworkAttachmentAddrs(
			instance.Namespace, instance.Spec.NetworkAttachments, instance.Status.NetworkAttachments)

		// Configure target IP addresses using all addresses in the network
		// attachments. This relies on the fact that the LVM backend can only
		// have one replica.
		if len(networkAttachmentAddrs) > 0 {
			templateParameters["TargetIpAddress"] = networkAttachmentAddrs[0]
			if len(networkAttachmentAddrs) > 1 {
				templateParameters["TargetSecondaryIpAddresses"] = strings.Join(networkAttachmentAddrs[1:], ",")
			}
		}
	}

	configTemplates := []util.Template{
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
func (r *CinderVolumeReconciler) createHashOfInputHashes(
	ctx context.Context,
	instance *cinderv1beta1.CinderVolume,
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

// processCustomServiceConfig - A general purpose hook that may extend the customServiceConfig
// string in the CR. Note that this is not equivalent to a transforming or defaulting webhook,
// as the CR is not updated. It only influences the final settings in cinder.conf.
//
// Currently this is limited to defining the list of enabled_backends in case its missing.
// The function also returns a boolean that indicates whether the LVM driver is being used.
func processCustomServiceConfig(customServiceConfig string) (bool, string) {
	svcConfigLines := strings.Split(customServiceConfig, "\n")
	hasEnabledBackends := false
	usesLVM := false
	numDrivers := 0
	defaultSectionIdx := -1
	sectionName := ""
	backendNames := []string{}

	for idx, rawLine := range svcConfigLines {
		line := strings.TrimSpace(rawLine)
		token := strings.TrimSpace(strings.SplitN(line, "=", 2)[0])

		if token == "" || strings.HasPrefix(token, "#") {
			// Skip blank lines and comments
			continue
		}

		if token == "enabled_backends" {
			// Note when the CR already specifies the enabled_backends
			hasEnabledBackends = true

		} else if token == "[DEFAULT]" {
			// Note when the customServiceConfig contains a [DEFAULT] section
			defaultSectionIdx = idx

		} else if strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]") {
			// Note the section name before looking for a volume_backend_name
			sectionName = strings.Trim(token, "[]")

		} else if token == "volume_backend_name" {
			// The section name is used in the list of enabled_backends
			backendNames = append(backendNames, sectionName)

		} else if token == "volume_driver" {
			numDrivers += 1
			if strings.HasSuffix(line, ".LVMVolumeDriver") {
				usesLVM = true
			}
		}
	}

	// Account for the fact that LVM is the default driver. If the backend's driver
	// is implicitly LVM, the number of explicitly configured drivers will be less
	// than the number of backends.
	if numDrivers < len(backendNames) {
		usesLVM = true
	}

	var extendedConfig string
	if hasEnabledBackends || len(backendNames) == 0 {
		// Nothing to do, just return the original customServiceConfig
		extendedConfig = customServiceConfig

	} else if defaultSectionIdx == -1 {
		// Prepend a new [DEFAULT] section that specifies the enabled_backends
		extendedConfig = fmt.Sprintf(
			"[DEFAULT]\nenabled_backends=%s\n%s",
			strings.Join(backendNames, ","),
			customServiceConfig)

	} else {
		// Replace the "[DEFAULT]" line in svcConfigLines with text that includes the enabled_backends
		svcConfigLines[defaultSectionIdx] = fmt.Sprintf(
			"[DEFAULT]\nenabled_backends=%s",
			strings.Join(backendNames, ","))
		extendedConfig = strings.Join(svcConfigLines, "\n")
	}

	return usesLVM, extendedConfig
}
