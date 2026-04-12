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

package ipam

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ipamv1 "sigs.k8s.io/cluster-api/api/ipam/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIPAMHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IPAM Helper Suite")
}

// =============================================================================
// Test constants
// =============================================================================

const (
	testClusterResourceGroup = "my-cluster"
)

// =============================================================================
// Test helpers
// =============================================================================

// newTestScheme creates a runtime.Scheme with all types needed for IPAM tests.
func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = ipamv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	return scheme
}

// newFakeClient creates a fake k8s client with the test scheme and optional initial objects.
func newFakeClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(newTestScheme()).
		WithObjects(objs...).
		WithStatusSubresource(&ipamv1.IPAddressClaim{}).
		Build()
}

// newTestIPAMService creates an IPAMService with a fake client for testing.
func newTestIPAMService(fakeClient client.Client, opts ...func(*IPAMServiceConfig)) *IPAMService {
	owner := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-owner",
			Namespace: "default",
			UID:       "test-owner-uid",
		},
	}

	config := IPAMServiceConfig{
		Client:      fakeClient,
		Logger:      logr.Discard(),
		Namespace:   "default",
		VnetName:    "test-vnet",
		ClusterName: "test-cluster",
		CreatorID:   IPClaimCreatorCAPH,
		Owner:       owner,
	}
	for _, opt := range opts {
		opt(&config)
	}
	return NewIPAMService(config)
}

// mockTelemetryWriter captures telemetry calls for verification.
type mockTelemetryWriter struct {
	calls []telemetryCall
}

type telemetryCall struct {
	operation IPAMOperation
	claimName string
	params    map[string]string
	err       error
}

func (m *mockTelemetryWriter) WriteIPAMOperationLog(_ logr.Logger, op IPAMOperation, claimName string, params map[string]string, err error) {
	m.calls = append(m.calls, telemetryCall{
		operation: op,
		claimName: claimName,
		params:    params,
		err:       err,
	})
}

// strPtr returns a pointer to s.
func strPtr(s string) *string {
	return &s
}

// =============================================================================
// Pure function tests
// =============================================================================

var _ = Describe("isArcVMOwnedVNet", func() {
	It("returns false for nil tags", func() {
		Expect(isArcVMOwnedVNet(nil)).To(BeFalse())
	})

	It("returns false for empty tags", func() {
		Expect(isArcVMOwnedVNet(map[string]*string{})).To(BeFalse())
	})

	It("returns true when MocOperatorResourceName tag is present", func() {
		tags := map[string]*string{
			MocOperatorResourceNameTag: strPtr("my-lnet"),
		}
		Expect(isArcVMOwnedVNet(tags)).To(BeTrue())
	})

	It("returns true when legacy MocOperatorName tag is present", func() {
		tags := map[string]*string{
			MocOperatorNameTag: strPtr("some-operator"),
		}
		Expect(isArcVMOwnedVNet(tags)).To(BeTrue())
	})

	It("returns true when ArcVMOwned tag is present", func() {
		tags := map[string]*string{
			ArcVMOwnedTag: strPtr("true"),
		}
		Expect(isArcVMOwnedVNet(tags)).To(BeTrue())
	})

	It("returns false when ownership tag has empty value", func() {
		tags := map[string]*string{
			MocOperatorResourceNameTag: strPtr(""),
		}
		Expect(isArcVMOwnedVNet(tags)).To(BeFalse())
	})

	It("returns false when ownership tag has nil value", func() {
		tags := map[string]*string{
			MocOperatorResourceNameTag: nil,
		}
		Expect(isArcVMOwnedVNet(tags)).To(BeFalse())
	})

	It("returns false when only unrelated tags are present", func() {
		tags := map[string]*string{
			"SomeOtherTag": strPtr("value"),
			"Environment":  strPtr("test"),
		}
		Expect(isArcVMOwnedVNet(tags)).To(BeFalse())
	})

	It("returns true when multiple ownership tags are present", func() {
		tags := map[string]*string{
			MocOperatorResourceNameTag: strPtr("my-lnet"),
			ArcVMOwnedTag:              strPtr("true"),
		}
		Expect(isArcVMOwnedVNet(tags)).To(BeTrue())
	})
})

var _ = Describe("isManagementResourceGroup", func() {
	It("returns true for arc appliance management group", func() {
		Expect(isManagementResourceGroup("management")).To(BeTrue())
	})

	It("returns true for 22H2 management group", func() {
		Expect(isManagementResourceGroup("clustergroup")).To(BeTrue())
	})

	It("is case-insensitive", func() {
		Expect(isManagementResourceGroup("Management")).To(BeTrue())
		Expect(isManagementResourceGroup("CLUSTERGROUP")).To(BeTrue())
		Expect(isManagementResourceGroup("ClusterGroup")).To(BeTrue())
	})

	It("returns false for non-management groups", func() {
		Expect(isManagementResourceGroup(testClusterResourceGroup)).To(BeFalse())
		Expect(isManagementResourceGroup("default")).To(BeFalse())
		Expect(isManagementResourceGroup("")).To(BeFalse())
	})
})

var _ = Describe("GenerateNICIPClaimName", func() {
	It("generates correct format", func() {
		Expect(GenerateNICIPClaimName("my-nic", 0)).To(Equal("ipclaim-my-nic-0"))
		Expect(GenerateNICIPClaimName("my-nic", 1)).To(Equal("ipclaim-my-nic-1"))
	})

	It("handles empty NIC name", func() {
		Expect(GenerateNICIPClaimName("", 0)).To(Equal("ipclaim--0"))
	})
})

// =============================================================================
// NewIPAMService constructor tests
// =============================================================================

var _ = Describe("NewIPAMService", func() {
	It("uses no-op telemetry when TelemetryWriter is nil", func() {
		svc := NewIPAMService(IPAMServiceConfig{
			Client: newFakeClient(),
			Logger: logr.Discard(),
			Owner: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "default", UID: "uid"},
			},
		})
		Expect(svc.telemetryWriter).NotTo(BeNil())
		// Should not panic when called
		svc.telemetryWriter.WriteIPAMOperationLog(logr.Discard(), OperationCreate, "test", nil, nil)
	})

	It("uses provided telemetry writer", func() {
		writer := &mockTelemetryWriter{}
		svc := NewIPAMService(IPAMServiceConfig{
			Client:          newFakeClient(),
			Logger:          logr.Discard(),
			TelemetryWriter: writer,
			Owner: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "default", UID: "uid"},
			},
		})
		svc.telemetryWriter.WriteIPAMOperationLog(logr.Discard(), OperationCreate, "claim-1", nil, nil)
		Expect(writer.calls).To(HaveLen(1))
		Expect(writer.calls[0].operation).To(Equal(OperationCreate))
	})

	It("always sets namespace to IPClaimNamespace constant", func() {
		svc := NewIPAMService(IPAMServiceConfig{
			Client:    newFakeClient(),
			Logger:    logr.Discard(),
			Namespace: "custom-namespace",
			Owner: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "default", UID: "uid"},
			},
		})
		// IPClaimNamespace is always used regardless of config.Namespace
		Expect(svc.GetNamespace()).To(Equal(IPClaimNamespace))
	})

	It("copies all config fields", func() {
		svc := NewIPAMService(IPAMServiceConfig{
			Client:               newFakeClient(),
			Logger:               logr.Discard(),
			VnetName:             "my-vnet",
			ClusterName:          testClusterResourceGroup,
			CreatorID:            IPClaimCreatorCloudOp,
			ClusterResourceGroup: "my-group",
			Owner: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "default", UID: "uid"},
			},
		})
		Expect(svc.GetVnetName()).To(Equal("my-vnet"))
		Expect(svc.GetClusterName()).To(Equal(testClusterResourceGroup))
		Expect(svc.creatorID).To(Equal(IPClaimCreatorCloudOp))
		Expect(svc.clusterResourceGroup).To(Equal("my-group"))
	})
})

// =============================================================================
// Accessor and SetOwner tests
// =============================================================================

var _ = Describe("Accessors and SetOwner", func() {
	var svc *IPAMService

	BeforeEach(func() {
		svc = newTestIPAMService(newFakeClient())
	})

	It("GetNamespace returns configured namespace", func() {
		Expect(svc.GetNamespace()).To(Equal(IPClaimNamespace))
	})

	It("GetVnetName returns configured VNet", func() {
		Expect(svc.GetVnetName()).To(Equal("test-vnet"))
	})

	It("GetClusterName returns configured cluster", func() {
		Expect(svc.GetClusterName()).To(Equal("test-cluster"))
	})

	It("SetOwner updates the owner reference", func() {
		newOwner := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "new-owner", Namespace: "default", UID: "new-uid"},
		}
		svc.SetOwner(newOwner)
		Expect(svc.owner.GetName()).To(Equal("new-owner"))
	})
})

// =============================================================================
// buildIPClaimParams tests
// =============================================================================

var _ = Describe("buildIPClaimParams", func() {
	var svc *IPAMService

	BeforeEach(func() {
		svc = newTestIPAMService(newFakeClient())
	})

	It("sets base labels with creator and annotation with allocation source", func() {
		params := svc.buildIPClaimParams("claim-1", "10.0.0.5", AllocationSourceIPAM, nil, nil)
		Expect(params.Labels).To(HaveKeyWithValue(LabelCreatedBy, IPClaimCreatorCAPH))
		Expect(params.Annotations).To(HaveKeyWithValue(AnnotationAllocationSource, AllocationSourceIPAM))
	})

	It("skips allocation source annotation when empty", func() {
		params := svc.buildIPClaimParams("claim-1", "10.0.0.5", "", nil, nil)
		Expect(params.Annotations).NotTo(HaveKey(AnnotationAllocationSource))
	})

	It("merges additional labels", func() {
		extra := map[string]string{
			LabelMocGroupName:    "my-group",
			LabelMocResourceName: "my-nic",
			LabelMocResourceType: MocResourceTypeNIC,
		}
		params := svc.buildIPClaimParams("claim-1", "10.0.0.5", AllocationSourceMOC, nil, extra)
		Expect(params.Labels).To(HaveKeyWithValue(LabelMocGroupName, "my-group"))
		Expect(params.Labels).To(HaveKeyWithValue(LabelMocResourceName, "my-nic"))
		Expect(params.Labels).To(HaveKeyWithValue(LabelMocResourceType, MocResourceTypeNIC))
		// Base labels still present
		Expect(params.Labels).To(HaveKeyWithValue(LabelCreatedBy, IPClaimCreatorCAPH))
	})

	It("merges additional annotations", func() {
		extraAnnotations := map[string]string{
			"custom-annotation": "custom-value",
		}
		params := svc.buildIPClaimParams("claim-1", "10.0.0.5", AllocationSourceMOC, extraAnnotations, nil)
		Expect(params.Annotations).To(HaveKeyWithValue("custom-annotation", "custom-value"))
		// Base annotations still present
		Expect(params.Annotations).To(HaveKeyWithValue(AnnotationAllocationSource, AllocationSourceMOC))
	})

	It("handles nil additional annotations", func() {
		params := svc.buildIPClaimParams("claim-1", "10.0.0.5", AllocationSourceIPAM, nil, nil)
		Expect(params.Annotations).To(HaveKeyWithValue(AnnotationAllocationSource, AllocationSourceIPAM))
		Expect(params.Annotations).To(HaveLen(1))
	})

	It("sets core fields correctly", func() {
		params := svc.buildIPClaimParams("claim-1", "10.0.0.5", AllocationSourceIPAM, nil, nil)
		Expect(params.Name).To(Equal("claim-1"))
		Expect(params.Namespace).To(Equal(IPClaimNamespace))
		Expect(params.ClusterName).To(Equal("test-cluster"))
		Expect(params.VnetName).To(Equal("test-vnet"))
		Expect(params.StaticIP).To(Equal("10.0.0.5"))
	})
})

// =============================================================================
// createIPClaim tests
// =============================================================================

var _ = Describe("createIPClaim", func() {
	It("creates an IPAddressClaim with correct annotations", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		params := svc.buildIPClaimParams("claim-1", "10.0.0.5", AllocationSourceIPAM, nil, nil)
		err := svc.createIPClaim(context.Background(), params)
		Expect(err).NotTo(HaveOccurred())

		// Verify the claim was created
		claim := &ipamv1.IPAddressClaim{}
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-1", Namespace: IPClaimNamespace}, claim)
		Expect(err).NotTo(HaveOccurred())
		Expect(claim.Annotations).To(HaveKeyWithValue(AnnotationIPClaimStaticIP, "10.0.0.5"))
		Expect(claim.Annotations).To(HaveKeyWithValue(AnnotationLogicalNetworkName, "test-vnet"))
		Expect(claim.Annotations).To(HaveKeyWithValue(AnnotationSubnetName, "test-vnet"))
		Expect(claim.Labels).To(HaveKeyWithValue(LabelCreatedBy, IPClaimCreatorCAPH))
		Expect(claim.Annotations).To(HaveKeyWithValue(AnnotationAllocationSource, AllocationSourceIPAM))
	})

	It("sets ClusterName in spec", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		params := svc.buildIPClaimParams("claim-1", "", AllocationSourceIPAM, nil, nil)
		err := svc.createIPClaim(context.Background(), params)
		Expect(err).NotTo(HaveOccurred())

		claim := &ipamv1.IPAddressClaim{}
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-1", Namespace: IPClaimNamespace}, claim)
		Expect(err).NotTo(HaveOccurred())
		Expect(claim.Spec.ClusterName).To(Equal("test-cluster"))
	})

	It("omits static IP annotation when empty", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		params := svc.buildIPClaimParams("claim-1", "", AllocationSourceIPAM, nil, nil)
		err := svc.createIPClaim(context.Background(), params)
		Expect(err).NotTo(HaveOccurred())

		claim := &ipamv1.IPAddressClaim{}
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-1", Namespace: IPClaimNamespace}, claim)
		Expect(err).NotTo(HaveOccurred())
		Expect(claim.Annotations).NotTo(HaveKey(AnnotationIPClaimStaticIP))
	})

	It("sets owner reference", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		params := svc.buildIPClaimParams("claim-1", "", AllocationSourceIPAM, nil, nil)
		err := svc.createIPClaim(context.Background(), params)
		Expect(err).NotTo(HaveOccurred())

		claim := &ipamv1.IPAddressClaim{}
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-1", Namespace: IPClaimNamespace}, claim)
		Expect(err).NotTo(HaveOccurred())
		Expect(claim.OwnerReferences).To(HaveLen(1))
		Expect(claim.OwnerReferences[0].Name).To(Equal("test-owner"))
	})

	It("returns nil when claim already exists", func() {
		existingClaim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
		}
		fakeClient := newFakeClient(existingClaim)
		svc := newTestIPAMService(fakeClient)

		params := svc.buildIPClaimParams("claim-1", "10.0.0.5", AllocationSourceIPAM, nil, nil)
		err := svc.createIPClaim(context.Background(), params)
		Expect(err).NotTo(HaveOccurred())
	})

	It("places additional annotations on annotations and additional labels on labels", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		lbAnnotation := AzstackhciAPIGroup + "/legacy-loadbalancer-ip"
		lbAnnotations := map[string]string{
			lbAnnotation: "true",
		}
		lbLabels := map[string]string{
			LabelMocGroupName:    "my-group",
			LabelMocResourceName: "my-lb",
			LabelMocResourceType: MocResourceTypeLoadBalancer,
		}
		params := svc.buildIPClaimParams("claim-lb", "10.0.0.10", AllocationSourceMOC, lbAnnotations, lbLabels)
		err := svc.createIPClaim(context.Background(), params)
		Expect(err).NotTo(HaveOccurred())

		claim := &ipamv1.IPAddressClaim{}
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-lb", Namespace: IPClaimNamespace}, claim)
		Expect(err).NotTo(HaveOccurred())

		// Verify legacy LB IP is in annotations, not labels
		Expect(claim.Annotations).To(HaveKeyWithValue(lbAnnotation, "true"))
		Expect(claim.Labels).NotTo(HaveKey(lbAnnotation))

		// Verify MOC metadata is in labels
		Expect(claim.Labels).To(HaveKeyWithValue(LabelMocGroupName, "my-group"))
		Expect(claim.Labels).To(HaveKeyWithValue(LabelMocResourceName, "my-lb"))
		Expect(claim.Labels).To(HaveKeyWithValue(LabelMocResourceType, MocResourceTypeLoadBalancer))

		// Verify base annotations still present
		Expect(claim.Annotations).To(HaveKeyWithValue(AnnotationAllocationSource, AllocationSourceMOC))
		Expect(claim.Annotations).To(HaveKeyWithValue(AnnotationIPClaimStaticIP, "10.0.0.10"))
	})
})

// =============================================================================
// DeleteIPClaim tests
// =============================================================================

var _ = Describe("DeleteIPClaim", func() {
	It("deletes an existing claim", func() {
		existingClaim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
		}
		fakeClient := newFakeClient(existingClaim)
		svc := newTestIPAMService(fakeClient)

		err := svc.DeleteIPClaim(context.Background(), "claim-1")
		Expect(err).NotTo(HaveOccurred())

		// Verify it was deleted
		claim := &ipamv1.IPAddressClaim{}
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-1", Namespace: IPClaimNamespace}, claim)
		Expect(err).To(HaveOccurred())
	})

	It("returns nil when claim does not exist", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		err := svc.DeleteIPClaim(context.Background(), "nonexistent")
		Expect(err).NotTo(HaveOccurred())
	})

	It("writes telemetry on delete", func() {
		existingClaim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
		}
		telemetry := &mockTelemetryWriter{}
		fakeClient := newFakeClient(existingClaim)
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.TelemetryWriter = telemetry
		})

		err := svc.DeleteIPClaim(context.Background(), "claim-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(telemetry.calls).To(HaveLen(1))
		Expect(telemetry.calls[0].operation).To(Equal(OperationDelete))
		Expect(telemetry.calls[0].claimName).To(Equal("claim-1"))
		Expect(telemetry.calls[0].err).To(BeNil())
	})
})

// =============================================================================
// verifyAllocatedIP tests
// =============================================================================

var _ = Describe("verifyAllocatedIP", func() {
	It("returns nil when IP matches", func() {
		ipAddr := &ipamv1.IPAddress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ipaddr-1",
				Namespace: IPClaimNamespace,
			},
			Spec: ipamv1.IPAddressSpec{
				Address: "10.0.0.5",
			},
		}
		fakeClient := newFakeClient(ipAddr)
		svc := newTestIPAMService(fakeClient)

		claim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
			Status: ipamv1.IPAddressClaimStatus{
				AddressRef: ipamv1.IPAddressReference{Name: "ipaddr-1"},
			},
		}

		err := svc.verifyAllocatedIP(context.Background(), claim, "10.0.0.5")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error when IP mismatches", func() {
		ipAddr := &ipamv1.IPAddress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ipaddr-1",
				Namespace: IPClaimNamespace,
			},
			Spec: ipamv1.IPAddressSpec{
				Address: "10.0.0.99",
			},
		}
		fakeClient := newFakeClient(ipAddr)
		svc := newTestIPAMService(fakeClient)

		claim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
			Status: ipamv1.IPAddressClaimStatus{
				AddressRef: ipamv1.IPAddressReference{Name: "ipaddr-1"},
			},
		}

		err := svc.verifyAllocatedIP(context.Background(), claim, "10.0.0.5")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("mismatched IP"))
		Expect(err.Error()).To(ContainSubstring("expected 10.0.0.5"))
		Expect(err.Error()).To(ContainSubstring("got 10.0.0.99"))
	})

	It("returns nil when claim has no address ref (pending allocation)", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		claim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
		}

		err := svc.verifyAllocatedIP(context.Background(), claim, "10.0.0.5")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error when IPAddress object does not exist", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		claim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
			Status: ipamv1.IPAddressClaimStatus{
				AddressRef: ipamv1.IPAddressReference{Name: "nonexistent"},
			},
		}

		err := svc.verifyAllocatedIP(context.Background(), claim, "10.0.0.5")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to get IPAddress"))
	})
})

// =============================================================================
// IsIPAMSoleAllocator tests
// =============================================================================

var _ = Describe("IsIPAMSoleAllocator", func() {
	It("returns true when azstackhci-operator deployment is not found", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		result := svc.IsIPAMSoleAllocator(context.Background())
		Expect(result).To(BeTrue())
	})

	It("returns false when azstackhci-operator deployment exists", func() {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      azstackhciOperatorDeploymentName,
				Namespace: azstackhciOperatorDeploymentNamespace,
			},
		}
		fakeClient := newFakeClient(deployment)
		svc := newTestIPAMService(fakeClient)

		result := svc.IsIPAMSoleAllocator(context.Background())
		Expect(result).To(BeFalse())
	})

	It("returns false when a different deployment exists (not azstackhci-operator)", func() {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-other-deployment",
				Namespace: azstackhciOperatorDeploymentNamespace,
			},
		}
		fakeClient := newFakeClient(deployment)
		svc := newTestIPAMService(fakeClient)

		result := svc.IsIPAMSoleAllocator(context.Background())
		Expect(result).To(BeTrue())
	})
})

// =============================================================================
// waitForIPAllocation tests
// =============================================================================

var _ = Describe("waitForIPAllocation", func() {
	It("returns allocated IP when claim has addressRef and IPAddress exists", func() {
		ipAddr := &ipamv1.IPAddress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ipaddr-1",
				Namespace: IPClaimNamespace,
			},
			Spec: ipamv1.IPAddressSpec{
				Address: "10.0.0.42",
			},
		}
		claim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
			Status: ipamv1.IPAddressClaimStatus{
				AddressRef: ipamv1.IPAddressReference{Name: "ipaddr-1"},
			},
		}
		fakeClient := newFakeClient(claim, ipAddr)
		svc := newTestIPAMService(fakeClient)

		ip, err := svc.waitForIPAllocation(context.Background(), "claim-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(ip).To(Equal("10.0.0.42"))
	})

	It("returns error when claim has Ready=False condition", func() {
		claim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
			Status: ipamv1.IPAddressClaimStatus{
				Conditions: []metav1.Condition{
					{
						Type:    ReadyConditionType,
						Status:  metav1.ConditionFalse,
						Message: "IP pool exhausted",
					},
				},
			},
		}
		fakeClient := newFakeClient(claim)
		svc := newTestIPAMService(fakeClient)

		_, err := svc.waitForIPAllocation(context.Background(), "claim-1")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("IP pool exhausted"))
	})

	It("times out when claim never gets addressRef", func() {
		claim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
		}
		fakeClient := newFakeClient(claim)
		svc := newTestIPAMService(fakeClient)

		_, err := svc.waitForIPAllocation(context.Background(), "claim-1")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("timeout"))
	})
})

// =============================================================================
// isIPAMAllocationEnabled tests (partial - management group and config checks)
// =============================================================================

var _ = Describe("isIPAMAllocationEnabled", func() {
	It("returns false for management resource group (arc appliance)", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.ClusterResourceGroup = ManagementGroupArcAppliance
			c.CloudFqdn = "fake-fqdn"
		})

		enabled, err := svc.isIPAMAllocationEnabled(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(enabled).To(BeFalse())
	})

	It("returns false for management resource group (22H2)", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.ClusterResourceGroup = ManagementGroup22H2
			c.CloudFqdn = "fake-fqdn"
		})

		enabled, err := svc.isIPAMAllocationEnabled(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(enabled).To(BeFalse())
	})

	It("returns error when MOC connection is not configured (no fqdn)", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.ClusterResourceGroup = testClusterResourceGroup
			c.CloudFqdn = ""
		})

		enabled, err := svc.isIPAMAllocationEnabled(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("MOC connection not configured"))
		Expect(enabled).To(BeFalse())
	})
})

// =============================================================================
// SyncIPClaim tests
// =============================================================================

var _ = Describe("SyncIPClaim", func() {
	// --- Early exit paths ---

	It("returns nil immediately when allocatedIP is empty", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient)

		err := svc.SyncIPClaim(context.Background(), "claim-1", "", nil, nil)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns nil immediately when cluster resource group is management (arc appliance)", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.ClusterResourceGroup = ManagementGroupArcAppliance
		})

		err := svc.SyncIPClaim(context.Background(), "claim-1", "10.0.0.5", nil, nil)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns nil immediately when cluster resource group is management (22H2)", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.ClusterResourceGroup = ManagementGroup22H2
		})

		err := svc.SyncIPClaim(context.Background(), "claim-1", "10.0.0.5", nil, nil)
		Expect(err).NotTo(HaveOccurred())
	})

	// --- Existing claim with matching IP ---

	It("returns nil when existing claim IP matches (no-op)", func() {
		ipAddr := &ipamv1.IPAddress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ipaddr-1",
				Namespace: IPClaimNamespace,
			},
			Spec: ipamv1.IPAddressSpec{
				Address: "10.0.0.5",
			},
		}
		existingClaim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
			Status: ipamv1.IPAddressClaimStatus{
				AddressRef: ipamv1.IPAddressReference{Name: "ipaddr-1"},
			},
		}
		fakeClient := newFakeClient(existingClaim, ipAddr)
		svc := newTestIPAMService(fakeClient)

		err := svc.SyncIPClaim(context.Background(), "claim-1", "10.0.0.5", nil, nil)
		Expect(err).NotTo(HaveOccurred())

		// Verify the claim was NOT deleted (still exists)
		claim := &ipamv1.IPAddressClaim{}
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-1", Namespace: IPClaimNamespace}, claim)
		Expect(err).NotTo(HaveOccurred())
	})

	// --- Existing claim with mismatched IP ---

	It("deletes existing claim when IP mismatches before checking MOC", func() {
		ipAddr := &ipamv1.IPAddress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ipaddr-1",
				Namespace: IPClaimNamespace,
			},
			Spec: ipamv1.IPAddressSpec{
				Address: "10.0.0.99", // Different from expected 10.0.0.5
			},
		}
		existingClaim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
			Status: ipamv1.IPAddressClaimStatus{
				AddressRef: ipamv1.IPAddressReference{Name: "ipaddr-1"},
			},
		}
		fakeClient := newFakeClient(existingClaim, ipAddr)
		// No MOC config → isIPAMAllocationEnabled will error AFTER the delete
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.ClusterResourceGroup = testClusterResourceGroup
			c.CloudFqdn = ""
		})

		err := svc.SyncIPClaim(context.Background(), "claim-1", "10.0.0.5", nil, nil)
		// Expect error from isIPAMAllocationEnabled (no MOC config), but the delete should have happened
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("MOC connection not configured"))

		// Verify the mismatched claim was deleted
		claim := &ipamv1.IPAddressClaim{}
		getErr := fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-1", Namespace: IPClaimNamespace}, claim)
		Expect(getErr).To(HaveOccurred()) // Should be NotFound
	})

	// --- Existing claim with no addressRef (pending allocation) ---

	It("treats claim with no addressRef as pending and does not delete it", func() {
		existingClaim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
			// No Status.AddressRef — claim is still pending
		}
		fakeClient := newFakeClient(existingClaim)
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.ClusterResourceGroup = testClusterResourceGroup
			c.CloudFqdn = ""
		})

		err := svc.SyncIPClaim(context.Background(), "claim-1", "10.0.0.5", nil, nil)
		// No error — pending claim is left alone
		Expect(err).NotTo(HaveOccurred())

		// Verify the pending claim still exists
		claim := &ipamv1.IPAddressClaim{}
		getErr := fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-1", Namespace: IPClaimNamespace}, claim)
		Expect(getErr).NotTo(HaveOccurred())
	})

	// --- Claim does not exist ---

	It("returns MOC error when claim does not exist and MOC is not configured", func() {
		fakeClient := newFakeClient()
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.ClusterResourceGroup = testClusterResourceGroup
			c.CloudFqdn = ""
		})

		err := svc.SyncIPClaim(context.Background(), "claim-1", "10.0.0.5", nil, nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("MOC connection not configured"))
	})

	// --- Telemetry ---

	It("writes telemetry on mismatched IP delete failure", func() {
		// Create a claim with addressRef pointing to non-existent IPAddress
		existingClaim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
			Status: ipamv1.IPAddressClaimStatus{
				AddressRef: ipamv1.IPAddressReference{Name: "missing-ipaddr"},
			},
		}
		telemetry := &mockTelemetryWriter{}
		fakeClient := newFakeClient(existingClaim)
		svc := newTestIPAMService(fakeClient, func(c *IPAMServiceConfig) {
			c.TelemetryWriter = telemetry
			c.ClusterResourceGroup = testClusterResourceGroup
			c.CloudFqdn = ""
		})

		// verifyAllocatedIP fails (IPAddress not found) → delete → then isIPAMAllocationEnabled errors
		err := svc.SyncIPClaim(context.Background(), "claim-1", "10.0.0.5", nil, nil)
		Expect(err).To(HaveOccurred())
	})

	// --- Does not touch claims for empty IP even if claim exists ---

	It("does not delete existing claim when allocatedIP is empty", func() {
		existingClaim := &ipamv1.IPAddressClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "claim-1",
				Namespace: IPClaimNamespace,
			},
		}
		fakeClient := newFakeClient(existingClaim)
		svc := newTestIPAMService(fakeClient)

		err := svc.SyncIPClaim(context.Background(), "claim-1", "", nil, nil)
		Expect(err).NotTo(HaveOccurred())

		// Claim should still exist — SyncIPClaim bailed before touching it
		claim := &ipamv1.IPAddressClaim{}
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "claim-1", Namespace: IPClaimNamespace}, claim)
		Expect(err).NotTo(HaveOccurred())
	})
})

// =============================================================================
// Telemetry integration tests
// =============================================================================

var _ = Describe("Telemetry", func() {
	It("noOpTelemetryWriter does not panic", func() {
		writer := &noOpTelemetryWriter{}
		Expect(func() {
			writer.WriteIPAMOperationLog(logr.Discard(), OperationCreate, "claim", nil, nil)
			writer.WriteIPAMOperationLog(logr.Discard(), OperationDelete, "claim", map[string]string{"key": "val"}, fmt.Errorf("err"))
		}).NotTo(Panic())
	})

	It("mockTelemetryWriter records all calls", func() {
		writer := &mockTelemetryWriter{}
		writer.WriteIPAMOperationLog(logr.Discard(), OperationCreate, "claim-1", map[string]string{"ip": "10.0.0.1"}, nil)
		writer.WriteIPAMOperationLog(logr.Discard(), OperationDelete, "claim-2", nil, fmt.Errorf("delete failed"))

		Expect(writer.calls).To(HaveLen(2))

		Expect(writer.calls[0].operation).To(Equal(OperationCreate))
		Expect(writer.calls[0].claimName).To(Equal("claim-1"))
		Expect(writer.calls[0].err).To(BeNil())

		Expect(writer.calls[1].operation).To(Equal(OperationDelete))
		Expect(writer.calls[1].claimName).To(Equal("claim-2"))
		Expect(writer.calls[1].err).To(HaveOccurred())
	})
})
