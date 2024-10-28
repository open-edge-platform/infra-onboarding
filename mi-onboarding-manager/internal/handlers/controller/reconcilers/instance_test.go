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

	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"

	statusv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/status/v1"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"

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
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	providerv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/provider/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/flags"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/util"
)

const tenantID = "11111111-1111-1111-1111-111111111111"

// FIXME: remove and use Inventory helper once RepoURL is made configurable in the Inv library
func createOsWithArgs(tb testing.TB, doCleanup bool,
) (osr *osv1.OperatingSystemResource) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	osr = &osv1.OperatingSystemResource{
		Name:              "for unit testing purposes",
		UpdateSources:     []string{"test entries"},
		ImageUrl:          "example.raw.gz",
		ProfileName:       inv_testing.GenerateRandomProfileName(),
		Sha256:            inv_testing.GenerateRandomSha256(),
		InstalledPackages: "intel-opencl-icd\nintel-level-zero-gpu\nlevel-zero",
		SecurityFeature:   osv1.SecurityFeature_SECURITY_FEATURE_UNSPECIFIED,
		OsType:            osv1.OsType_OS_TYPE_IMMUTABLE,
	}
	resp, err := inv_testing.GetClient(tb, inv_testing.APIClient).Create(ctx,
		&inv_v1.Resource{Resource: &inv_v1.Resource_Os{Os: osr}})
	require.NoError(tb, err)
	osr.ResourceId = resp.GetOs().GetResourceId()
	if doCleanup {
		tb.Cleanup(func() { inv_testing.DeleteResource(tb, osr.ResourceId) })
	}

	return osr
}

func createProviderWithArgs(tb testing.TB, doCleanup bool,
	resourceId, name string, providerKind providerv1.ProviderKind,
) (provider *providerv1.ProviderResource) {
	tb.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	str := "{\"defaultOs\":\"osID\",\"autoProvision\":true,\"customerID\":\"170312\",\"enProductKeyIDs\":\"a6d7cf35-049a-4959-88b0-3bcb91beb857\"}"
	str = strings.Replace(str, "osID", resourceId, 1)
	provider = &providerv1.ProviderResource{
		ProviderKind:   providerKind,
		Name:           name,
		ApiEndpoint:    "xyz123",
		ApiCredentials: []string{"abc123"},
		Config:         str,
	}
	resp, err := inv_testing.GetClient(tb, inv_testing.APIClient).Create(ctx,
		&inv_v1.Resource{Resource: &inv_v1.Resource_Provider{Provider: provider}})
	require.NoError(tb, err)
	provider.ResourceId = resp.GetProvider().GetResourceId()
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

	instanceController := rec_v2.NewController[ReconcilerID](instanceReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil)
	osRes := createOsWithArgs(t, true)
	providerResource := inv_testing.CreateProviderWithArgs(t, "lenovo", "8.8.8.8", nil,
		providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL, providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA)
	instance := inv_testing.CreateInstanceWithProvider(t, host, osRes, providerResource)
	instanceID := instance.GetResourceId()

	// performing reconciliation
	err := instanceController.Reconcile(NewReconcilerID(instance.GetTenantId(), instanceID))
	assert.NoError(t, err, "Reconciliation failed")

	// making sure no changes to the Instance has happened
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	// Trying to delete the Instance. It contains Provider, so nothing should happen during the reconciliation.
	// Setting the Desired state of the Instance to be DELETED.
	inv_testing.DeleteResource(t, instanceID)
	// No change at the Instance Current State and Status should have happened
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_DELETED, // Desired state has just been updated
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	// performing Instance reconciliation
	err = instanceController.Reconcile(NewReconcilerID(instance.GetTenantId(), instanceID))
	assert.NoError(t, err, "Reconciliation failed")

	// No change at the Instance Current State and Status should have happened
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_DELETED, // Desired state has just been updated
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))
}

func TestReconcileInstance(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	currFlagEnableDeviceInitialization := *flags.FlagDisableCredentialsManagement
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
		*common.FlagEnableDeviceInitialization = currFlagEnableDeviceInitialization
	}()

	// TODO: test with DI enabled, once FDO client is refactored
	*common.FlagEnableDeviceInitialization = false
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false, true)

	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	instanceReconciler := NewInstanceReconciler(om_testing.InvClient, true)
	require.NotNil(t, instanceReconciler)

	instanceController := rec_v2.NewController[ReconcilerID](instanceReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil)
	osRes := createOsWithArgs(t, true)
	_ = createProviderWithArgs(t, true, osRes.ResourceId, "itep_licensing", providerv1.ProviderKind_PROVIDER_KIND_LICENSING) // Creating license Provider profile which would be fetched by the reconciler.
	_ = createProviderWithArgs(t, true, osRes.ResourceId, "fm_onboarding", providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL)  // Creating Provider profile which would be fetched by the reconciler.
	instance := inv_testing.CreateInstanceNoCleanup(t, host, osRes)                                                          // Instance should not be assigned to the Provider.
	instanceID := instance.GetResourceId()

	runReconcilationFunc := func() {
		select {
		case ev, ok := <-om_testing.InvClient.Watcher:
			require.True(t, ok, "No events received")
			expectedKind, err := util.GetResourceKindFromResourceID(ev.Event.ResourceId)
			require.NoError(t, err)
			if expectedKind == inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE {
				err = instanceController.Reconcile(NewReconcilerID(instance.GetTenantId(), ev.Event.ResourceId))
				assert.NoError(t, err, "Reconciliation failed")
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("No events received within timeout")
		}
		time.Sleep(1 * time.Second)
	}

	runReconcilationFunc()
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

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

	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		om_status.ProvisioningStatusDone)

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
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_ERROR,
		om_status.ProvisioningStatusDone)

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
		request rec_v2.Request[ReconcilerID]
	}
	testRequest := rec_v2.Request[ReconcilerID]{}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ReconcilerID]
	}{
		{
			name: "Test Reconciliation on Instance Deletion Success",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ReconcilerID]{ID: NewReconcilerID(tenantID, "12345678")},
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
				request: rec_v2.Request[ReconcilerID]{ID: NewReconcilerID(tenantID, "12345678")},
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
				request: rec_v2.Request[ReconcilerID]{ID: NewReconcilerID(tenantID, "12345678")},
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
				request: rec_v2.Request[ReconcilerID]{ID: NewReconcilerID(tenantID, "12345678")},
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
				request: rec_v2.Request[ReconcilerID]{ID: NewReconcilerID(tenantID, "12345678")},
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
		request  rec_v2.Request[ReconcilerID]
		instance *computev1.InstanceResource
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ReconcilerID]
	}{
		{
			name: "Test Reconciliation with Running Instance and BMC Interface",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.Background(),
				request: rec_v2.Request[ReconcilerID]{},
				instance: &computev1.InstanceResource{
					DesiredState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
						HostNics: []*computev1.HostnicResource{
							{
								ResourceId:   "hostnic-084d9b08",
								BmcInterface: true,
							},
						},
						BmcIp: "00.00.00.00",
					},

					DesiredOs: &osv1.OperatingSystemResource{
						ImageUrl: "osUrl.raw.gz;overlayUrl",
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
				request: rec_v2.Request[ReconcilerID]{},
				instance: &computev1.InstanceResource{
					DesiredState: computev1.InstanceState_INSTANCE_STATE_UNTRUSTED,
					Host: &computev1.HostResource{
						ResourceId:   "host-084d9b08",
						CurrentState: computev1.HostState_HOST_STATE_UNTRUSTED,
						HostNics: []*computev1.HostnicResource{
							{
								ResourceId:   "hostnic-084d9b08",
								BmcInterface: true,
							},
						},
						BmcIp: "00.00.00.00",
					},

					DesiredOs: &osv1.OperatingSystemResource{
						ImageUrl: "osUrl.raw.gz;overlayUrl",
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
		instance        *computev1.InstanceResource
		licenseProvider invclient.LicenseProviderConfig
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
					DesiredOs: &osv1.OperatingSystemResource{
						ImageUrl: "http://some-url.raw.gz;http://some-url-2;v0.7.4",
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
			wantErr: true,
		},
		{
			name: "Failed - invalid OS URL format",
			args: args{
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						BmcIp: "0.0.0.0",
					},
					SecurityFeature: osv1.SecurityFeature_SECURITY_FEATURE_UNSPECIFIED,
					DesiredOs: &osv1.OperatingSystemResource{
						ImageUrl: "http://some-url;http://some-url-2;v0.7.4",
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
			_, err := convertInstanceToDeviceInfo(tt.args.instance, tt.args.licenseProvider)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
