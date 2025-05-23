// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package reconcilers_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	statusv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/status/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	inv_status "github.com/open-edge-platform/infra-core/inventory/v2/pkg/status"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/handlers/controller/reconcilers"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/invclient"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	om_testing "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/testing"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell"
	om_status "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/status"
	rec_v2 "github.com/open-edge-platform/orch-library/go/pkg/controller/v2"
)

const tenantID = "11111111-1111-1111-1111-111111111111"

func getMD5Hash(text string) string {
	hasher := sha256.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getFirstNChars(hash string, n int) string {
	if len(hash) < n {
		return hash
	}
	return hash[:n]
}

// FIXME: remove and use Inventory helper once RepoURL is made configurable in the Inv library.
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
		OsProvider:        osv1.OsProviderKind_OS_PROVIDER_KIND_INFRA,
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
	resourceID, name string, providerKind providerv1.ProviderKind,
) (provider *providerv1.ProviderResource) { //nolint:unparam // current tests do not use, future tests may
	tb.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	str := "{\"defaultOs\":\"osID\",\"autoProvision\":true}"
	str = strings.Replace(str, "osID", resourceID, 1)
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
// In this case, no reconciliation should be performed for such Instance
// (the reconciliation should happen in the Provider-specific RM,e.g., LOC-A RM).
func TestReconcileInstanceWithProvider(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	instanceReconciler := reconcilers.NewInstanceReconciler(om_testing.InvClient, true)
	require.NotNil(t, instanceReconciler)

	instanceController := rec_v2.NewController[reconcilers.ReconcilerID](instanceReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil)
	osRes := createOsWithArgs(t, true)
	providerResource := inv_testing.CreateProviderWithArgs(t, "lenovo", "8.8.8.8", nil,
		providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL, providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA)
	instance := inv_testing.CreateInstanceWithProvider(t, host, osRes, providerResource)
	instanceID := instance.GetResourceId()

	// performing reconciliation
	err := instanceController.Reconcile(reconcilers.NewReconcilerID(instance.GetTenantId(), instanceID))
	assert.NoError(t, err, "Reconciliation failed")

	// making sure no changes to the Instance has happened
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New(inv_status.DefaultProvisioningStatus, statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))
	// Trying to delete the Instance. It contains Provider, so nothing should happen during the reconciliation.
	// Setting the Desired state of the Instance to be DELETED.
	inv_testing.DeleteResource(t, instanceID)
	// No change at the Instance Current State and Status should have happened
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_DELETED, // Desired state has just been updated
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New(inv_status.DefaultProvisioningStatus, statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	// performing Instance reconciliation
	err = instanceController.Reconcile(reconcilers.NewReconcilerID(instance.GetTenantId(), instanceID))
	assert.NoError(t, err, "Reconciliation failed")

	// No change at the Instance Current State and Status should have happened
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_DELETED, // Desired state has just been updated
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New(inv_status.DefaultProvisioningStatus, statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))
}

func TestReconcileInstanceNonEIM(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	instanceReconciler := reconcilers.NewInstanceReconciler(om_testing.InvClient, true)
	require.NotNil(t, instanceReconciler)

	instanceController := rec_v2.NewController[reconcilers.ReconcilerID](instanceReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil)
	osRes := inv_testing.CreateOsWithOpts(t, true, func(osr *osv1.OperatingSystemResource) {
		osr.ProfileName = inv_testing.GenerateRandomProfileName()
		osr.Sha256 = inv_testing.GenerateRandomSha256()
		osr.OsType = osv1.OsType_OS_TYPE_MUTABLE
		osr.OsProvider = osv1.OsProviderKind_OS_PROVIDER_KIND_LENOVO
	})
	instance := inv_testing.CreateInstance(t, host, osRes) // Instance should not be assigned to the Provider.
	instanceID := instance.GetResourceId()

	runReconcilationFuncInstance(t, instanceController, instance)

	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New(inv_status.DefaultProvisioningStatus, statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

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

	runReconcilationFuncInstance(t, instanceController, instance)

	// Instance should not move into the RUNNING state as OS provisioning should be skipped
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New(inv_status.DefaultProvisioningStatus, statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))
}

//nolint:funlen // it's a test
func TestReconcileInstance(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
	}()

	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false, true)

	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	instanceReconciler := reconcilers.NewInstanceReconciler(om_testing.InvClient, true)
	require.NotNil(t, instanceReconciler)

	instanceController := rec_v2.NewController[reconcilers.ReconcilerID](instanceReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil)

	osRes := createOsWithArgs(t, true)
	// Creating Provider profile which would be fetched by the reconciler.
	_ = createProviderWithArgs(t, true, osRes.ResourceId, onboarding_types.DefaultProviderName,
		providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL)
	// Instance should not be assigned to the Provider.
	instance := inv_testing.CreateInstanceNoCleanup(t, host, osRes)
	instanceID := instance.GetResourceId()

	runReconcilationFuncInstance(t, instanceController, instance)
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New(inv_status.DefaultProvisioningStatus, statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

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

	runReconcilationFuncInstance(t, instanceController, instance)

	// Host is still not onboarded, so instance provisioning should not start.
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New(inv_status.DefaultProvisioningStatus, statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	// Set host current state to ONBOARDED
	res = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				ResourceId:   host.GetResourceId(),
				CurrentState: computev1.HostState_HOST_STATE_ONBOARDED,
			},
		},
	}
	_, err = inv_testing.TestClients[inv_testing.RMClient].Update(ctx, host.GetResourceId(),
		&fieldmaskpb.FieldMask{Paths: []string{computev1.HostResourceFieldCurrentState}}, res)
	require.NoError(t, err)

	// This case is handled with internal events, let's mock it by calling the reconciler again
	err = instanceController.Reconcile(reconcilers.NewReconcilerID(instance.GetTenantId(), instanceID))
	assert.NoError(t, err, "Reconciliation failed for internal events")

	// Wait for reconciler to do its magic.
	time.Sleep(500 * time.Millisecond)

	// getting rid of the Host event
	<-om_testing.InvClient.Watcher

	// Now provisioning should have happened
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		om_status.ProvisioningStatusDone)

	// run again, current_state == desired_state
	runReconcilationFuncInstance(t, instanceController, instance)

	// report error status
	res = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId:                  instanceID,
				ProvisioningStatus:          om_status.ProvisioningStatusFailed.Status,
				ProvisioningStatusIndicator: om_status.ProvisioningStatusFailed.StatusIndicator,
			},
		},
	}

	fmk = fieldmaskpb.FieldMask{Paths: []string{
		computev1.InstanceResourceFieldProvisioningStatus,
		computev1.InstanceResourceFieldProvisioningStatusIndicator,
	}}
	_, err = inv_testing.TestClients[inv_testing.RMClient].Update(ctx, instanceID, &fmk, res)
	require.NoError(t, err)

	// state shall be not change, but status is ERROR
	runReconcilationFuncInstance(t, instanceController, instance)
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		om_status.ProvisioningStatusFailed)

	// delete
	res = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId:   instanceID,
				DesiredState: computev1.InstanceState_INSTANCE_STATE_DELETED,
			},
		},
	}
	fmk = fieldmaskpb.FieldMask{Paths: []string{computev1.InstanceResourceFieldDesiredState}}
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(ctx, instanceID, &fmk, res)
	require.NoError(t, err)
	zlogInst := logging.GetLogger("InstanceReconciler")
	zlogInst.Debug().Msgf("Instance %s updated to DELETED", instanceID)

	runReconcilationFuncInstance(t, instanceController, instance)

	_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, instanceID)
	require.True(t, inv_errors.IsNotFound(err))
}

//nolint:funlen // it's a test
func TestReconcileInstanceHostDeauthorized(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
	}()

	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false, true)

	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	instanceReconciler := reconcilers.NewInstanceReconciler(om_testing.InvClient, true)
	require.NotNil(t, instanceReconciler)

	instanceController := rec_v2.NewController[reconcilers.ReconcilerID](instanceReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil)

	osRes := createOsWithArgs(t, false)
	// Creating Provider profile which would be fetched by the reconciler.
	_ = createProviderWithArgs(t, false, osRes.ResourceId, onboarding_types.DefaultProviderName+"_test",
		providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL)
	// Instance should not be assigned to the Provider.
	instance := inv_testing.CreateInstanceNoCleanup(t, host, osRes)
	instanceID := instance.GetResourceId()

	runReconcilationFuncInstance(t, instanceController, instance)
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

	runReconcilationFuncInstance(t, instanceController, instance)

	// Host is still not onboarded, so instance provisioning should not start.
	om_testing.AssertInstance(t, instance.GetTenantId(), instanceID,
		computev1.InstanceState_INSTANCE_STATE_RUNNING,
		computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	// Set host current state to UNTRUSTED to trigger deauthorized flow
	res = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				ResourceId:   host.GetResourceId(),
				CurrentState: computev1.HostState_HOST_STATE_UNTRUSTED,
			},
		},
	}
	_, err = inv_testing.TestClients[inv_testing.RMClient].Update(ctx, host.GetResourceId(),
		&fieldmaskpb.FieldMask{Paths: []string{computev1.HostResourceFieldCurrentState}}, res)
	require.NoError(t, err)

	fmk = fieldmaskpb.FieldMask{Paths: []string{computev1.InstanceResourceFieldProvisioningStatusIndicator}}
	res = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId:                  instanceID,
				ProvisioningStatusIndicator: statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS,
			},
		},
	}
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(ctx, instanceID, &fmk, res)
	require.NoError(t, err)

	runReconcilationFuncInstance(t, instanceController, instance)

	// This case is handled with internal events, let's mock it by calling the reconciler again
	err = instanceController.Reconcile(reconcilers.NewReconcilerID(instance.GetTenantId(), instanceID))
	assert.NoError(t, err, "Reconciliation failed for internal events")

	// Wait for reconciler to do its magic.
	time.Sleep(500 * time.Millisecond)

	// getting rid of the Host event
	<-om_testing.InvClient.Watcher

	// Now instance statuses should be updated to Unknown since Host is deauthorized
	om_testing.AssertInstanceStatuses(t, instance.GetTenantId(), instanceID,
		om_status.InstanceStatusUnknown, om_status.ProvisioningStatusUnknown,
		om_status.UpdateStatusUnknown, om_status.TrustedAttestationStatusUnknown)

	// delete
	res = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId:   instanceID,
				DesiredState: computev1.InstanceState_INSTANCE_STATE_DELETED,
			},
		},
	}
	fmk = fieldmaskpb.FieldMask{Paths: []string{computev1.InstanceResourceFieldDesiredState}}
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(ctx, instanceID, &fmk, res)
	require.NoError(t, err)

	runReconcilationFuncInstance(t, instanceController, instance)

	_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, instanceID)
	require.True(t, inv_errors.IsNotFound(err))
}

func runReconcilationFuncInstance(t *testing.T, instanceController *rec_v2.Controller[reconcilers.ReconcilerID],
	instance *computev1.InstanceResource,
) {
	t.Helper()
	runReconcilationFunc := func() {
		select {
		case ev, ok := <-om_testing.InvClient.Watcher:
			require.True(t, ok, "No events received")
			expectedKind, err := util.GetResourceKindFromResourceID(ev.Event.ResourceId)
			require.NoError(t, err)
			if expectedKind == inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE {
				err = instanceController.Reconcile(reconcilers.NewReconcilerID(instance.GetTenantId(), ev.Event.ResourceId))
				assert.NoError(t, err, "Reconciliation failed")
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("No events received within timeout")
		}
		time.Sleep(1 * time.Second)
	}
	runReconcilationFunc()
}

func TestNewInstanceReconciler(t *testing.T) {
	type args struct {
		c *invclient.OnboardingInventoryClient
	}
	tests := []struct {
		name string
		args args
		want *reconcilers.InstanceReconciler
	}{
		{
			name: "Positive -for InstanceReconciler",
			args: args{
				c: &invclient.OnboardingInventoryClient{},
			},
			want: reconcilers.NewInstanceReconciler(&invclient.OnboardingInventoryClient{}, false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reconcilers.NewInstanceReconciler(tt.args.c, false); !reflect.DeepEqual(got, tt.want) {
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
		request rec_v2.Request[reconcilers.ReconcilerID]
	}
	testRequest := rec_v2.Request[reconcilers.ReconcilerID]{}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[reconcilers.ReconcilerID]
	}{
		{
			name: "Test Reconciliation on Instance Deletion Success",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[reconcilers.ReconcilerID]{ID: reconcilers.NewReconcilerID(tenantID, "12345678")},
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
				request: rec_v2.Request[reconcilers.ReconcilerID]{ID: reconcilers.NewReconcilerID(tenantID, "12345678")},
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
				request: rec_v2.Request[reconcilers.ReconcilerID]{ID: reconcilers.NewReconcilerID(tenantID, "12345678")},
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
				request: rec_v2.Request[reconcilers.ReconcilerID]{ID: reconcilers.NewReconcilerID(tenantID, "12345678")},
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
				request: rec_v2.Request[reconcilers.ReconcilerID]{ID: reconcilers.NewReconcilerID(tenantID, "12345678")},
			},
			want: testRequest.Ack(),
		},
	}
	originalDir, getwdErr := os.Getwd()
	if getwdErr != nil {
		t.Fatalf("Failed to  working directory : %v", getwdErr)
	}
	err := os.Chdir(originalDir)
	if err != nil {
		t.Fatalf("Failed to change working directory back to original: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ir := reconcilers.NewInstanceReconciler(tt.fields.invClient, false)
			if got := ir.Reconcile(tt.args.ctx, tt.args.request); reflect.DeepEqual(got, tt.want) {
				t.Errorf("InstanceReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileInstanceWithDesiredUpdateState(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	instanceReconciler := reconcilers.NewInstanceReconciler(om_testing.InvClient, true)
	require.NotNil(t, instanceReconciler)

	instanceController := rec_v2.NewController[reconcilers.ReconcilerID](instanceReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil)
	osRes := createOsWithArgs(t, true)
	providerResource := inv_testing.CreateProviderWithArgs(t, "lenovo", "8.8.8.8", nil,
		providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL, providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA)
	instance := inv_testing.CreateInstanceWithProvider(t, host, osRes, providerResource)
	instanceID := instance.GetResourceId()
	resDesired := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId:   instanceID,
				DesiredState: computev1.InstanceState_INSTANCE_STATE_UNTRUSTED,
			},
		},
	}
	fmkDesired := fieldmaskpb.FieldMask{Paths: []string{computev1.InstanceResourceFieldDesiredState}}
	_, err := inv_testing.TestClients[inv_testing.APIClient].Update(context.Background(), instanceID, &fmkDesired, resDesired)
	require.NoError(t, err)
	err = instanceController.Reconcile(reconcilers.NewReconcilerID(instance.GetTenantId(), instanceID))
	assert.NoError(t, err, "Reconciliation failed")
}

func FuzzTestReconcile(f *testing.F) {
	// Add initial seed data
	f.Add("0809039")

	f.Fuzz(func(t *testing.T, resourceID string) {
		om_testing.CreateInventoryOnboardingClientForTesting()
		t.Cleanup(func() {
			fmt.Println("Deleting Inventory Onboarding Client for Testing")
			om_testing.DeleteInventoryOnboardingClientForTesting()
		})

		// Log the resourceID being tested
		fmt.Printf("Testing with resourceID: %s", resourceID)

		if resourceID == "" {
			t.Skip("Skipping test because resourceID is empty")
			return
		}
		rID := "host-" + getFirstNChars(getMD5Hash(resourceID), 8)
		testRequest := rec_v2.Request[reconcilers.ReconcilerID]{}

		// ir := &reconcilers.InstanceReconciler{
		ir := reconcilers.NewInstanceReconciler(om_testing.InvClient, true)
		//}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		request := rec_v2.Request[reconcilers.ReconcilerID]{ID: reconcilers.NewReconcilerID(tenantID, rID)}
		got := ir.Reconcile(ctx, request)
		if reflect.DeepEqual(got, testRequest.Ack()) {
			t.Errorf("Fuzz Test InstanceReconciler.Reconcile() = %v", got)
		}
	})
}
