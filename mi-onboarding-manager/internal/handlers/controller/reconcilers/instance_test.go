// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	rec_v2 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.lib-go/pkg/controller/v2"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/tinkerbell"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	providerv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/provider/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
)

// FIXME: remove and use Inventory helper once RepoURL is made configurable in the Inv library
func createOsWithArgs(tb testing.TB, doCleanup bool,
) (osr *osv1.OperatingSystemResource) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	osr = &osv1.OperatingSystemResource{
		Name:              "for unit testing purposes",
		UpdateSources:     []string{"test entries"},
		RepoUrl:           "example.raw.gz",
		ProfileName:       inv_testing.GenerateRandomProfileName(),
		Sha256:            inv_testing.GenerateRandomSha256(),
		InstalledPackages: "intel-opencl-icd\nintel-level-zero-gpu\nlevel-zero",
		SecurityFeature:   osv1.SecurityFeature_SECURITY_FEATURE_UNSPECIFIED,
	}
	resp, err := inv_testing.GetClient(tb, inv_testing.APIClient).Create(ctx,
		&inv_v1.Resource{Resource: &inv_v1.Resource_Os{Os: osr}})
	require.NoError(tb, err)
	osr.ResourceId = resp.ResourceId
	if doCleanup {
		tb.Cleanup(func() { inv_testing.DeleteResource(tb, osr.ResourceId) })
	}

	return osr
}

func createProviderWithArgs(tb testing.TB, doCleanup bool,
	resourceId string) (provider *providerv1.ProviderResource) {
	tb.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	str := "{\"defaultOs\":\"osID\",\"autoProvision\":true,\"customerID\":\"170312\",\"enProductKeyIDs\":\"a6d7cf35-049a-4959-88b0-3bcb91beb857\"}"
	str = strings.Replace(str, "osID", resourceId, 1)
	provider = &providerv1.ProviderResource{
		ProviderKind:   providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL,
		Name:           "fm_onboarding",
		ApiEndpoint:    "xyz123",
		ApiCredentials: []string{"abc123"},
		Config:         str,
	}
	resp, err := inv_testing.GetClient(tb, inv_testing.APIClient).Create(ctx,
		&inv_v1.Resource{Resource: &inv_v1.Resource_Provider{Provider: provider}})
	require.NoError(tb, err)
	provider.ResourceId = resp.ResourceId
	if doCleanup {
		tb.Cleanup(func() { inv_testing.DeleteResource(tb, provider.ResourceId) })
	}
	return provider
}

// This TC verifies the case, when an event with Instance with pre-defined custom Provider (e.g., Lenovo) is obtained.
// In this case, no reconciliation should be performed for such Instance (the reconciliation should happen in the Provider-specific RM,
// e.g., LOC-A RM).
func TestReconcileInstanceWithProvider(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	instanceReconciler := NewInstanceReconciler(om_testing.InvClient, true)
	require.NotNil(t, instanceReconciler)

	instanceController := rec_v2.NewController[ResourceID](instanceReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	osRes := createOsWithArgs(t, true)
	providerResource := inv_testing.CreateProviderWithArgs(t, "lenovo", "8.8.8.8", nil,
		providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL, providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA)
	instance := inv_testing.CreateInstanceWithProvider(t, host, osRes, providerResource)
	instanceID := instance.GetResourceId()

	// performing reconciliation
	err := instanceController.Reconcile(ResourceID(instanceID))
	assert.NoError(t, err, "Reconciliation failed")

	// making sure no changes to the Instance has happened
	om_testing.AssertInstance(t, instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		computev1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED)

	// Trying to delete the Instance. It contains Provider, so nothing should happen during the reconciliation.
	// Setting the Desired state of the Instance to be DELETED.
	inv_testing.DeleteResource(t, instanceID)
	// No change at the Instance Current State and Status should have happened
	om_testing.AssertInstance(t, instanceID,
		computev1.InstanceState_INSTANCE_STATE_DELETED, // Desired state has just been updated
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		computev1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED)

	// performing Instance reconciliation
	err = instanceController.Reconcile(ResourceID(instanceID))
	assert.NoError(t, err, "Reconciliation failed")

	// No change at the Instance Current State and Status should have happened
	om_testing.AssertInstance(t, instanceID,
		computev1.InstanceState_INSTANCE_STATE_DELETED, // Desired state has just been updated
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		computev1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED)
}

func TestReconcileInstance(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	currFlagEnableDeviceInitialization := *common.FlagDisableCredentialsManagement
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
		*common.FlagEnableDeviceInitialization = currFlagEnableDeviceInitialization
	}()

	// TODO: test with DI enabled, once FDO client is refactored
	*common.FlagEnableDeviceInitialization = false
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false)

	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	instanceReconciler := NewInstanceReconciler(om_testing.InvClient, true)
	require.NotNil(t, instanceReconciler)

	instanceController := rec_v2.NewController[ResourceID](instanceReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	osRes := createOsWithArgs(t, true)
	_ = createProviderWithArgs(t, true, osRes.ResourceId)           // Creating Provider profile which would be fetched by the reconciler.
	instance := inv_testing.CreateInstanceNoCleanup(t, host, osRes) // Instance should not be assigned to the Provider.
	instanceID := instance.GetResourceId()

	runReconcilationFunc := func() {
		select {
		case ev, ok := <-om_testing.InvClient.Watcher:
			require.True(t, ok, "No events received")
			expectedKind, err := util.GetResourceKindFromResourceID(ev.Event.ResourceId)
			require.NoError(t, err)
			if expectedKind == inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE {
				err = instanceController.Reconcile(ResourceID(ev.Event.ResourceId))
				assert.NoError(t, err, "Reconciliation failed")
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("No events received within timeout")
		}
		time.Sleep(1 * time.Second)
	}

	runReconcilationFunc()
	om_testing.AssertInstance(t, instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		computev1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED)

	// getting rid of the Host event
	<-om_testing.InvClient.Watcher

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// provision
	fmk := fieldmaskpb.FieldMask{Paths: []string{computev1.InstanceResourceFieldDesiredState}}
	res := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId:   instanceID,
				DesiredState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
			},
		},
	}
	_, err := inv_testing.TestClients[inv_testing.APIClient].Update(ctx, instanceID, &fmk, res)
	require.NoError(t, err)

	runReconcilationFunc()

	om_testing.AssertInstance(t, instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONED)

	// run again, current_state == desired_state
	runReconcilationFunc()

	// move into the error state
	res = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId:   instanceID,
				CurrentState: computev1.InstanceState_INSTANCE_STATE_ERROR,
			},
		},
	}
	_, err = inv_testing.TestClients[inv_testing.RMClient].Update(ctx, instanceID,
		&fieldmaskpb.FieldMask{Paths: []string{computev1.InstanceResourceFieldCurrentState}}, res)
	require.NoError(t, err)

	runReconcilationFunc()
	om_testing.AssertInstance(t, instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_ERROR,
		computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONED)

	// delete
	res = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId:   instanceID,
				DesiredState: computev1.InstanceState_INSTANCE_STATE_DELETED,
			},
		},
	}
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(ctx, instanceID, &fmk, res)
	require.NoError(t, err)

	runReconcilationFunc()

	_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, instanceID)
	require.True(t, inv_errors.IsNotFound(err))
}

func TestNewInstanceReconciler(t *testing.T) {
	type args struct {
		c *invclient.OnboardingInventoryClient
	}
	tests := []struct {
		name string
		args args
		want *InstanceReconciler
	}{
		{
			name: "Positive -for InstanceReconciler",
			args: args{
				c: &invclient.OnboardingInventoryClient{},
			},
			want: &InstanceReconciler{
				invClient: &invclient.OnboardingInventoryClient{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInstanceReconciler(tt.args.c, false); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInstanceReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInstanceReconciler_Reconcile(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ResourceID]
	}
	testRequest := rec_v2.Request[ResourceID]{}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ResourceID]
	}{
		{
			name: "Test Reconciliation on Instance Deletion Success",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
		{
			name: "Test Reconciliation on Instance Deletion Failure",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
		{
			name: "Test Reconciliation on Running Instance with Valid Host and OS",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
		{
			name: "Test Reconciliation on Running Instance with Invalid Host",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
		{
			name: "Test Reconciliation on Running Instance with Invalid OS",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
	}
	originalDir, _ := os.Getwd()
	err := os.Chdir(originalDir)
	if err != nil {
		t.Fatalf("Failed to change working directory back to original: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ir := &InstanceReconciler{
				invClient: tt.fields.invClient,
			}
			if got := ir.Reconcile(tt.args.ctx, tt.args.request); reflect.DeepEqual(got, tt.want) {
				t.Errorf("InstanceReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInstanceReconciler_reconcileInstance(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx      context.Context
		request  rec_v2.Request[ResourceID]
		instance *computev1.InstanceResource
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ResourceID]
	}{
		{
			name: "Test Reconciliation with Running Instance and BMC Interface",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.Background(),
				request: rec_v2.Request[ResourceID]{},
				instance: &computev1.InstanceResource{
					DesiredState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
					Host: &computev1.HostResource{
						ResourceId:       "host-084d9b08",
						LegacyHostStatus: computev1.HostStatus_HOST_STATUS_UNSPECIFIED,
						HostNics: []*computev1.HostnicResource{
							{
								ResourceId:   "hostnic-084d9b08",
								BmcInterface: true,
							},
						},
						BmcIp: "00.00.00.00",
					},

					Os: &osv1.OperatingSystemResource{
						RepoUrl: "osUrl.raw.gz;overlayUrl",
					},
				},
			},
		},
		{
			name: "Test Case for untrusted state",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.Background(),
				request: rec_v2.Request[ResourceID]{},
				instance: &computev1.InstanceResource{
					DesiredState: computev1.InstanceState_INSTANCE_STATE_UNTRUSTED,
					Host: &computev1.HostResource{
						ResourceId:       "host-084d9b08",
						LegacyHostStatus: computev1.HostStatus_HOST_STATUS_UNSPECIFIED,
						CurrentState: computev1.HostState_HOST_STATE_UNTRUSTED,
						HostNics: []*computev1.HostnicResource{
							{
								ResourceId:   "hostnic-084d9b08",
								BmcInterface: true,
							},
						},
						BmcIp: "00.00.00.00",
					},

					Os: &osv1.OperatingSystemResource{
						RepoUrl: "osUrl.raw.gz;overlayUrl",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ir := &InstanceReconciler{
				invClient: tt.fields.invClient,
			}
			if got := ir.reconcileInstance(tt.args.ctx, tt.args.request, tt.args.instance); reflect.DeepEqual(got, tt.want) {
				t.Errorf("InstanceReconciler.reconcileInstance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertInstanceToDeviceInfo(t *testing.T) {
	type args struct {
		instance *computev1.InstanceResource
		provider invclient.ProviderConfig
	}
	tests := []struct {
		name    string
		args    args
		want    utils.DeviceInfo
		wantErr bool
	}{
		{
			name: "Success",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						BmcIp: "0.0.0.0",
					},
					SecurityFeature: osv1.SecurityFeature_SECURITY_FEATURE_UNSPECIFIED,
					Os: &osv1.OperatingSystemResource{
						RepoUrl: "http://some-url.raw.gz;http://some-url-2;v0.7.4",
					},
				},
			},
			want: utils.DeviceInfo{
				OSImageURL:         "http://some-url.raw.gz",
				InstallerScriptURL: "http://some-url-2",
				TinkerVersion:      "v0.7.4",
				HwIP:               "0.0.0.0",
				Gateway:            "", // note that this is not valid and temporary
				Rootfspart:         "1",
				ClientImgName:      ClientImgName,
				ImgType:            "prod_bkc",
			},
			wantErr: false,
		},
		{
			name: "Failed - invalid OS URL format",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						BmcIp: "0.0.0.0",
					},
					SecurityFeature: osv1.SecurityFeature_SECURITY_FEATURE_UNSPECIFIED,
					Os: &osv1.OperatingSystemResource{
						RepoUrl: "http://some-url;http://some-url-2;v0.7.4",
					},
				},
			},
			want:    utils.DeviceInfo{},
			wantErr: true,
		},
		{
			name: "Failed - no OS resource associated",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						BmcIp: "000.0.0.0",
					},
					SecurityFeature: osv1.SecurityFeature_SECURITY_FEATURE_UNSPECIFIED,
				},
			},
			want:    utils.DeviceInfo{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env.ImgType = utils.ImgTypeBkc
			got, err := convertInstanceToDeviceInfo(tt.args.instance, tt.args.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

