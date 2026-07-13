/*
Copyright 2020 The Kubernetes Authors.
Portions Copyright © Microsoft Corporation.

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

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// TestFallbackVMRunningCondition covers the bug where a detailed VMRunningCondition (e.g.
// OutOfCapacityReason/MocUnreachableReason, copied moments earlier from the VM's own conditions)
// was being clobbered by a generic "VM state is unexpected" message whenever VMState was neither
// Succeeded nor Updating. See ADO 38717753. (Ported from PR #314 so this alternative-fix
// prototype isn't undermined by the same overwrite bug.)
func TestFallbackVMRunningCondition(t *testing.T) {
	const unexpectedState infrav1.VMState = "Failed"

	tests := []struct {
		name              string
		existingCondition *metav1.Condition
		wantReason        string
		wantMessage       string
	}{
		{
			name:              "no existing condition falls back to generic message",
			existingCondition: nil,
			wantReason:        infrav1.VMProvisionFailedReason,
			wantMessage:       `AzureStackHCI VM state "Failed" is unexpected`,
		},
		{
			name: "existing OutOfCapacity condition is preserved verbatim",
			existingCondition: &metav1.Condition{
				Type:    infrav1.VMRunningCondition,
				Status:  metav1.ConditionFalse,
				Reason:  infrav1.OutOfCapacityReason,
				Message: `OutOfCapacity: Location 'MocLocation' doesn't expose any nodes`,
			},
			wantReason:  infrav1.OutOfCapacityReason,
			wantMessage: `OutOfCapacity: Location 'MocLocation' doesn't expose any nodes`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fallbackVMRunningCondition(unexpectedState, tt.existingCondition)

			if got.Type != infrav1.VMRunningCondition {
				t.Errorf("Type = %q, want %q", got.Type, infrav1.VMRunningCondition)
			}
			if got.Status != metav1.ConditionFalse {
				t.Errorf("Status = %q, want %q", got.Status, metav1.ConditionFalse)
			}
			if got.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
			if got.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", got.Message, tt.wantMessage)
			}
		})
	}
}

// TestDeriveMachineReadyCondition covers the "alternative fix" prototype: mirroring
// VMRunningCondition verbatim into the standard CAPI Ready condition type. This is the function
// under investigation in AB#38717753's "alternative fix" track -- see the doc comment on
// deriveMachineReadyCondition for the open VMStateUpdating design question this test also exercises.
func TestDeriveMachineReadyCondition(t *testing.T) {
	tests := []struct {
		name      string
		vmRunning metav1.Condition
		want      metav1.Condition
	}{
		{
			name: "success is mirrored as Ready=True with empty message",
			vmRunning: metav1.Condition{
				Type:   infrav1.VMRunningCondition,
				Status: metav1.ConditionTrue,
				Reason: "VMRunning",
			},
			want: metav1.Condition{
				Type:   clusterv1.ReadyCondition,
				Status: metav1.ConditionTrue,
				Reason: "VMRunning",
			},
		},
		{
			name: "specific failure reason/message is mirrored verbatim (the actual fix payload)",
			vmRunning: metav1.Condition{
				Type:    infrav1.VMRunningCondition,
				Status:  metav1.ConditionFalse,
				Reason:  infrav1.OutOfCapacityReason,
				Message: `OutOfCapacity: Location 'MocLocation' doesn't expose any nodes`,
			},
			want: metav1.Condition{
				Type:    clusterv1.ReadyCondition,
				Status:  metav1.ConditionFalse,
				Reason:  infrav1.OutOfCapacityReason,
				Message: `OutOfCapacity: Location 'MocLocation' doesn't expose any nodes`,
			},
		},
		{
			// OPEN DESIGN QUESTION (see deriveMachineReadyCondition doc comment): this
			// documents CURRENT prototype behavior -- VMUpdating flips Ready=False, unlike
			// today's boolean-based CAPI fallback which stays True through routine updates.
			// This test exists to make that behavior explicit and easy to find/revisit, not
			// to assert it is the correct final design.
			name: "VMUpdating is mirrored as Ready=False (OPEN QUESTION -- may cause new flapping)",
			vmRunning: metav1.Condition{
				Type:   infrav1.VMRunningCondition,
				Status: metav1.ConditionFalse,
				Reason: "VMUpdating",
			},
			want: metav1.Condition{
				Type:   clusterv1.ReadyCondition,
				Status: metav1.ConditionFalse,
				Reason: "VMUpdating",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveMachineReadyCondition(tt.vmRunning)

			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.want.Status)
			}
			if got.Reason != tt.want.Reason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.want.Reason)
			}
			if got.Message != tt.want.Message {
				t.Errorf("Message = %q, want %q", got.Message, tt.want.Message)
			}
		})
	}
}
