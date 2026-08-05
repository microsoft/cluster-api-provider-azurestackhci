/*
Copyright 2026 The Kubernetes Authors.

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
	"testing"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta2"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
)

// TestSetVMProvisionFailure verifies that a terminal VM-provisioning failure is recorded on
// BOTH the legacy VMRunningCondition and the CAPI contract "Ready" condition. The "Ready"
// condition is what CAPI mirrors into Machine.InfrastructureReadyCondition; without it the real
// SKU/capacity error is dropped and the operation stalls with an empty timeout. See AB#38511842.
func TestSetVMProvisionFailure(t *testing.T) {
	g := NewWithT(t)

	const message = "rpc error: code = Unknown desc = InvalidVMSku: vm size Standard_D4s_v3 is not available"
	vmScope := &scope.VirtualMachineScope{
		AzureStackHCIVirtualMachine: &infrav1.AzureStackHCIVirtualMachine{},
	}

	setVMProvisionFailure(vmScope, infrav1.OutOfCapacityReason, message)

	// Legacy condition is still set (unchanged behavior).
	vmRunning := conditions.Get(vmScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition)
	g.Expect(vmRunning).ToNot(BeNil())
	g.Expect(vmRunning.Status).To(Equal(metav1.ConditionFalse))
	g.Expect(vmRunning.Reason).To(Equal(infrav1.OutOfCapacityReason))
	g.Expect(vmRunning.Message).To(Equal(message))

	// New: the CAPI contract "Ready" condition carries the real reason + message so CAPI mirrors
	// it into Machine.InfrastructureReadyCondition (instead of the generic provisioned fallback).
	ready := conditions.Get(vmScope.AzureStackHCIVirtualMachine, clusterv1.ReadyCondition)
	g.Expect(ready).ToNot(BeNil())
	g.Expect(ready.Status).To(Equal(metav1.ConditionFalse))
	g.Expect(ready.Reason).To(Equal(infrav1.OutOfCapacityReason))
	g.Expect(ready.Message).To(Equal(message))
}
