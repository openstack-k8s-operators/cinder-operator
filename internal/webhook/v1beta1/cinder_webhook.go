/*
Copyright 2025.

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

// Package v1beta1 implements webhook handlers for Cinder v1beta1 API resources.
package v1beta1

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	cinderv1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
)

var (
	// ErrInvalidCinderType is returned when the object is not a Cinder type
	ErrInvalidCinderType = errors.New("object is not a Cinder type")
)

// nolint:unused
// log is for logging in this package.
var cinderlog = logf.Log.WithName("cinder-resource")

// SetupCinderWebhookWithManager registers the webhook for Cinder in the manager.
func SetupCinderWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&cinderv1.Cinder{}).
		WithValidator(&CinderCustomValidator{}).
		WithDefaulter(&CinderCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-cinder-openstack-org-v1beta1-cinder,mutating=true,failurePolicy=fail,sideEffects=None,groups=cinder.openstack.org,resources=cinders,verbs=create;update,versions=v1beta1,name=mcinder-v1beta1.kb.io,admissionReviewVersions=v1

// CinderCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Cinder when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type CinderCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &CinderCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Cinder.
func (d *CinderCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	cinder, ok := obj.(*cinderv1.Cinder)

	if !ok {
		return fmt.Errorf("%w: expected a Cinder object but got %T", ErrInvalidCinderType, obj)
	}
	cinderlog.Info("Defaulting for Cinder", "name", cinder.GetName())

	// Call the Default method on the Cinder type
	cinder.Default()

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-cinder-openstack-org-v1beta1-cinder,mutating=false,failurePolicy=fail,sideEffects=None,groups=cinder.openstack.org,resources=cinders,verbs=create;update,versions=v1beta1,name=vcinder-v1beta1.kb.io,admissionReviewVersions=v1

// CinderCustomValidator struct is responsible for validating the Cinder resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type CinderCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &CinderCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Cinder.
func (v *CinderCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	cinder, ok := obj.(*cinderv1.Cinder)
	if !ok {
		return nil, fmt.Errorf("%w: expected a Cinder object but got %T", ErrInvalidCinderType, obj)
	}
	cinderlog.Info("Validation for Cinder upon creation", "name", cinder.GetName())

	// Call the ValidateCreate method on the Cinder type
	return cinder.ValidateCreate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Cinder.
func (v *CinderCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	cinder, ok := newObj.(*cinderv1.Cinder)
	if !ok {
		return nil, fmt.Errorf("%w: expected a Cinder object for the newObj but got %T", ErrInvalidCinderType, newObj)
	}
	cinderlog.Info("Validation for Cinder upon update", "name", cinder.GetName())

	// Call the ValidateUpdate method on the Cinder type
	return cinder.ValidateUpdate(oldObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Cinder.
func (v *CinderCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	cinder, ok := obj.(*cinderv1.Cinder)
	if !ok {
		return nil, fmt.Errorf("%w: expected a Cinder object but got %T", ErrInvalidCinderType, obj)
	}
	cinderlog.Info("Validation for Cinder upon deletion", "name", cinder.GetName())

	// Call the ValidateDelete method on the Cinder type
	return cinder.ValidateDelete()
}
