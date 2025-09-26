/*
Copyright 2024.

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

// Package controllers implements the cinder-operator Kubernetes controllers.
package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Common static errors for all cinder controllers
var (
	ErrNetworkAttachmentWaiting  = errors.New("network attachment waiting")
	ErrNetworkAttachmentNotFound = errors.New("network-attachment-definition not found")
	ErrStatefulSetWaiting        = errors.New("waiting for StatefulSet to start reconciling")
	ErrNetworkAttachmentConfig   = errors.New("not all pods have interfaces with ips as configured in NetworkAttachments")
)

type conditionUpdater interface {
	Set(c *condition.Condition)
	MarkTrue(t condition.Type, messageFormat string, messageArgs ...any)
}

type topologyHandler interface {
	GetSpecTopologyRef() *topologyv1.TopoRef
	GetLastAppliedTopology() *topologyv1.TopoRef
	SetLastAppliedTopology(t *topologyv1.TopoRef)
}

func ensureTopology(
	ctx context.Context,
	helper *helper.Helper,
	instance topologyHandler,
	finalizer string,
	conditionUpdater conditionUpdater,
	defaultLabelSelector metav1.LabelSelector,
) (*topologyv1.Topology, error) {

	topology, err := topologyv1.EnsureServiceTopology(
		ctx,
		helper,
		instance.GetSpecTopologyRef(),
		instance.GetLastAppliedTopology(),
		finalizer,
		defaultLabelSelector,
	)
	if err != nil {
		conditionUpdater.Set(condition.FalseCondition(
			condition.TopologyReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.TopologyReadyErrorMessage,
			err.Error()))
		return nil, fmt.Errorf("waiting for Topology requirements: %w", err)
	}
	// update the Status with the last retrieved Topology (or set it to nil)
	instance.SetLastAppliedTopology(instance.GetSpecTopologyRef())
	// update the Topology condition only when a Topology is referenced and has
	// been retrieved (err == nil)
	if tr := instance.GetSpecTopologyRef(); tr != nil {
		// update the TopologyRef associated condition
		conditionUpdater.MarkTrue(
			condition.TopologyReadyCondition,
			condition.TopologyReadyMessage,
		)
	}
	return topology, nil
}

// verifyServiceSecret - ensures that the Secret object exists and the expected
// fields are in the Secret. It also sets a hash of the values of the expected
// fields passed as input.
func verifyServiceSecret(
	ctx context.Context,
	secretName types.NamespacedName,
	expectedFields []string,
	reader client.Reader,
	conditionUpdater conditionUpdater,
	requeueTimeout time.Duration,
	envVars *map[string]env.Setter,
) (ctrl.Result, error) {

	hash, res, err := secret.VerifySecret(ctx, secretName, expectedFields, reader, requeueTimeout)
	if err != nil {
		conditionUpdater.Set(condition.FalseCondition(
			condition.InputReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.InputReadyErrorMessage,
			err.Error()))
		return res, err
	} else if (res != ctrl.Result{}) {
		// Since the service secret should have been manually created by the user and referenced in the spec,
		// we treat this as a warning because it means that the service will not be able to start.
		log.FromContext(ctx).Info(fmt.Sprintf("OpenStack secret %s not found", secretName))
		conditionUpdater.Set(condition.FalseCondition(
			condition.InputReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.InputReadyWaitingMessage))
		return res, nil
	}
	(*envVars)[secretName.Name] = env.SetValue(hash)
	return ctrl.Result{}, nil
}

// verifyConfigSecrets - It iterates over the secretNames passed as input and
// sets the hash of values in the envVars map.
func verifyConfigSecrets(
	ctx context.Context,
	h *helper.Helper,
	conditionUpdater conditionUpdater,
	secretNames []string,
	namespace string,
	envVars *map[string]env.Setter,
) (ctrl.Result, error) {
	var hash string
	var err error
	for _, secretName := range secretNames {
		// Do not try to GetSecret if the current element is empty
		if secretName == "" {
			continue
		}
		_, hash, err = secret.GetSecret(ctx, h, secretName, namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				// Since config secrets should have been previously automatically created by the parent
				// Cinder CR and then referenced in this instance's spec, we treat this as a warning
				// because it means that the service will not be able to start.
				log.FromContext(ctx).Info(fmt.Sprintf("Secret %s not found", secretName))
				conditionUpdater.Set(condition.FalseCondition(
					condition.InputReadyCondition,
					condition.ErrorReason,
					condition.SeverityWarning,
					condition.InputReadyWaitingMessage))
				return cinder.ResultRequeue, nil
			}
			conditionUpdater.Set(condition.FalseCondition(
				condition.InputReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.InputReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
		// Add a prefix to the var name to avoid accidental collision with other non-secret
		// vars. The secret names themselves will be unique.
		(*envVars)["secret-"+secretName] = env.SetValue(hash)
	}

	return ctrl.Result{}, nil
}
