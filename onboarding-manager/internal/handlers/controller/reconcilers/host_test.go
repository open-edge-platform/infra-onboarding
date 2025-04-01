// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
//
//nolint:testpackage // Keeping the test in the same package due to dependencies on unexported fields.
package reconcilers

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	statusv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/status/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/auth"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
	inv_status "github.com/open-edge-platform/infra-core/inventory/v2/pkg/status"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/invclient"
	om_testing "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/testing"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell"
	om_status "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/status"
	rec_v2 "github.com/open-edge-platform/orch-library/go/pkg/controller/v2"
)

func TestMain(m *testing.M) {
	*flags.FlagDisableCredentialsManagement = true
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(wd))))
	policyPath := projectRoot + "/out"
	migrationsDir := projectRoot + "/out"

	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false, true)

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestHostReconcileDeauthorization(t *testing.T) {
	currAuthServiceFactory := auth.AuthServiceFactory
	currFlagDisableCredentialsManagement := *flags.FlagDisableCredentialsManagement
	defer func() {
		auth.AuthServiceFactory = currAuthServiceFactory
		*flags.FlagDisableCredentialsManagement = currFlagDisableCredentialsManagement
	}()

	*flags.FlagDisableCredentialsManagement = false
	auth.AuthServiceFactory = om_testing.AuthServiceMockFactory(false, false, true)

	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	hostReconciler := NewHostReconciler(om_testing.InvClient, true)
	require.NotNil(t, hostReconciler)

	hostController := rec_v2.NewController[ReconcilerID](hostReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	host := inv_testing.CreateHost(t, nil, nil)

	hostID := host.GetResourceId()

	runReconcilationFunc := func() {
		select {
		case ev, ok := <-inv_testing.TestClientsEvents[inv_testing.RMClient]:
			require.True(t, ok, "No events received")
			err := hostController.Reconcile(NewReconcilerID(host.GetTenantId(), ev.Event.ResourceId))
			assert.NoError(t, err, "Reconciliation failed")
		case <-time.After(1 * time.Second):
			t.Fatalf("No events received within timeout")
		}
		time.Sleep(1 * time.Second)
	}

	runReconcilationFunc()
	om_testing.AssertHost(t, host.GetTenantId(), hostID,
		computev1.HostState_HOST_STATE_ONBOARDED,
		computev1.HostState_HOST_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				ResourceId:   hostID,
				DesiredState: computev1.HostState_HOST_STATE_UNTRUSTED,
			},
		},
	}
	fmk := fieldmaskpb.FieldMask{Paths: []string{computev1.HostResourceFieldDesiredState}}
	_, err := inv_testing.TestClients[inv_testing.APIClient].Update(ctx, hostID, &fmk, res)
	require.NoError(t, err)

	runReconcilationFunc()

	// auth service mock should return error, so no success
	om_testing.AssertHost(t, host.GetTenantId(), hostID,
		computev1.HostState_HOST_STATE_UNTRUSTED,
		computev1.HostState_HOST_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	auth.AuthServiceFactory = om_testing.AuthServiceMockFactory(false, false, false)

	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(ctx, hostID, &fmk, res)
	require.NoError(t, err)
	runReconcilationFunc()

	om_testing.AssertHost(t, host.GetTenantId(), hostID,
		computev1.HostState_HOST_STATE_UNTRUSTED,
		computev1.HostState_HOST_STATE_UNTRUSTED,
		om_status.AuthorizationStatusInvalidated)
}

func TestReconcileHostDeletion(t *testing.T) {
	currAuthServiceFactory := auth.AuthServiceFactory
	currFlagDisableCredentialsManagement := *flags.FlagDisableCredentialsManagement
	defer func() {
		auth.AuthServiceFactory = currAuthServiceFactory
		*flags.FlagDisableCredentialsManagement = currFlagDisableCredentialsManagement
	}()

	*flags.FlagDisableCredentialsManagement = false
	auth.AuthServiceFactory = om_testing.AuthServiceMockFactory(false, false, true)

	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	hostReconciler := NewHostReconciler(om_testing.InvClient, true)
	require.NotNil(t, hostReconciler)

	hostController := rec_v2.NewController[ReconcilerID](hostReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	hostSetup := createTestHostSetup(t)

	hostID := hostSetup.Host.GetResourceId()

	runReconcilationFunc(t, hostController, hostSetup.Host) // CREATED event
	om_testing.AssertHost(t, hostSetup.Host.GetTenantId(), hostID,
		computev1.HostState_HOST_STATE_ONBOARDED,
		computev1.HostState_HOST_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	// try to delete first, Instance exists so deletion should fail
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmk := fieldmaskpb.FieldMask{Paths: []string{computev1.HostResourceFieldDesiredState}}
	res := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				ResourceId:   hostID,
				DesiredState: computev1.HostState_HOST_STATE_DELETED,
			},
		},
	}
	_, err := inv_testing.TestClients[inv_testing.APIClient].Update(ctx, hostID, &fmk, res)
	require.NoError(t, err)

	runReconcilationFunc(t, hostController, hostSetup.Host) // UPDATED event (desired state to DELETED)
	runReconcilationFunc(t, hostController, hostSetup.Host) // UPDATED event (status update)

	expectedDetails := fmt.Sprintf("waiting on %s deletion", hostSetup.Instance.GetResourceId())
	om_testing.AssertHost(t, hostSetup.Host.GetTenantId(), hostID,
		computev1.HostState_HOST_STATE_DELETED,
		computev1.HostState_HOST_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))
	om_testing.AssertHostOnboardingStatus(t, hostID, om_status.ModernHostStatusDeletingWithDetails(expectedDetails))

	inv_testing.HardDeleteInstance(t, hostSetup.Instance.GetResourceId())

	// delete, attempt will fail but check if providerStatusDetail has changed
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: &computev1.HostResource{
				ResourceId:   hostID,
				DesiredState: computev1.HostState_HOST_STATE_DELETED,
			},
		},
	}
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(ctx, hostID, &fmk, res)
	require.NoError(t, err)

	runReconcilationFunc(t, hostController, hostSetup.Host)

	om_testing.AssertHost(t, hostSetup.Host.GetTenantId(), hostID,
		computev1.HostState_HOST_STATE_DELETED,
		computev1.HostState_HOST_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))
	om_testing.AssertHostOnboardingStatus(t, hostID, om_status.DeletingStatus)

	auth.AuthServiceFactory = om_testing.AuthServiceMockFactory(false, false, false)

	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(ctx, hostID, &fmk, res)
	require.NoError(t, err)

	runReconcilationFunc(t, hostController, hostSetup.Host)
	assertResourceNotFound(ctx, t,
		hostSetup.Host.GetResourceId(),
		hostSetup.HostNic.GetResourceId(),
		hostSetup.HostStorage.GetResourceId(),
		hostSetup.HostUsb.GetResourceId(),
		hostSetup.HostGpu.GetResourceId(),
		hostSetup.NicIP.GetResourceId(),
	)
}

func runReconcilationFunc(t *testing.T, hostController *rec_v2.Controller[ReconcilerID], host *computev1.HostResource) {
	t.Helper()
	runReconcilationFunc := func() {
		defer time.Sleep(1 * time.Second)
		for {
			select {
			case ev, ok := <-inv_testing.TestClientsEvents[inv_testing.RMClient]:
				require.True(t, ok, "No events received")
				resKind, err := util.GetResourceKindFromResourceID(ev.Event.ResourceId)
				require.NoError(t, err)
				if resKind != inv_v1.ResourceKind_RESOURCE_KIND_HOST {
					continue
				}
				err = hostController.Reconcile(NewReconcilerID(host.GetTenantId(), ev.Event.ResourceId))
				assert.NoError(t, err, "Reconciliation failed")
				return
			case <-time.After(1 * time.Second):
				t.Fatalf("No events received within timeout")
			}
		}
	}
	runReconcilationFunc()
}

type TestHostSetup struct {
	Host        *computev1.HostResource
	HostNic     *computev1.HostnicResource
	HostStorage *computev1.HoststorageResource
	HostUsb     *computev1.HostusbResource
	HostGpu     *computev1.HostgpuResource
	NicIP       *network_v1.IPAddressResource
	Instance    *computev1.InstanceResource
}

func createTestHostSetup(t *testing.T) *TestHostSetup {
	t.Helper()
	host := inv_testing.CreateHostNoCleanup(t, nil, nil)
	hostNic := inv_testing.CreateHostNicNoCleanup(t, host)
	hostStorage := inv_testing.CreateHostStorageNoCleanup(t, host)
	hostUsb := inv_testing.CreateHostusbNoCleanup(t, host)
	hostGpu := inv_testing.CreatHostGPUNoCleanup(t, host)
	nicIP := inv_testing.CreateIPAddress(t, hostNic, false)
	osRes := inv_testing.CreateOs(t)
	instance := inv_testing.CreateInstanceNoCleanup(t, host, osRes)

	return &TestHostSetup{
		Host:        host,
		HostNic:     hostNic,
		HostStorage: hostStorage,
		HostUsb:     hostUsb,
		HostGpu:     hostGpu,
		NicIP:       nicIP,
		Instance:    instance,
	}
}

func assertResourceNotFound(ctx context.Context, t *testing.T, resourceIDs ...string) {
	t.Helper()
	for _, resourceID := range resourceIDs {
		_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resourceID)
		require.True(t, inv_errors.IsNotFound(err), "Expected resource %s to be not found", resourceID)
	}
}

// This TC verifies the case, when an event with Host with pre-defined custom Provider (e.g., Lenovo) is obtained.
// In this case, no reconciliation should be performed for such Host(the reconciliation should happen in the Provider-specific RM
// e.g., LOC-A RM).
func TestReconcileHostWithProvider(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	hostReconciler := NewHostReconciler(om_testing.InvClient, true)
	require.NotNil(t, hostReconciler)

	hostController := rec_v2.NewController[ReconcilerID](hostReconciler.Reconcile, rec_v2.WithParallelism(1))
	// do not Stop() to avoid races, should be safe in tests

	// creating Provider
	providerResource := inv_testing.CreateProviderWithArgs(t, "lenovo", "8.8.8.8", nil,
		providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL, providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA)
	host := inv_testing.CreateHost(t, nil, providerResource)

	hostID := host.GetResourceId()

	// performing reconciliation
	err := hostController.Reconcile(NewReconcilerID(host.GetTenantId(), host.GetResourceId()))
	assert.NoError(t, err, "Reconciliation failed")

	om_testing.AssertHost(t, host.GetTenantId(), hostID,
		computev1.HostState_HOST_STATE_ONBOARDED,
		computev1.HostState_HOST_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	// Trying to delete the Host. It contains Provider, so nothing should happen during the reconciliation.
	// Setting the Desired state of the Host to be DELETED.
	inv_testing.DeleteResource(t, hostID)

	om_testing.AssertHost(t, host.GetTenantId(), hostID,
		computev1.HostState_HOST_STATE_DELETED,
		computev1.HostState_HOST_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))

	// performing reconciliation
	err = hostController.Reconcile(NewReconcilerID(host.GetTenantId(), host.GetResourceId()))
	assert.NoError(t, err, "Reconciliation failed")

	om_testing.AssertHost(t, host.GetTenantId(), hostID,
		computev1.HostState_HOST_STATE_DELETED,
		computev1.HostState_HOST_STATE_UNSPECIFIED,
		inv_status.New("", statusv1.StatusIndication_STATUS_INDICATION_UNSPECIFIED))
}

//nolint:dupl // These tests are for different reconcilers but have a similar structure.
func TestNewHostReconciler(t *testing.T) {
	type args struct {
		c *invclient.OnboardingInventoryClient
	}
	tests := []struct {
		name string
		args args
		want *HostReconciler
	}{
		{
			name: "Positive",
			args: args{
				c: &invclient.OnboardingInventoryClient{},
			},
			want: &HostReconciler{
				invClient: &invclient.OnboardingInventoryClient{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHostReconciler(tt.args.c, false); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHostReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostReconciler_Reconcile_Case1(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ReconcilerID]
	}
	testRequest := rec_v2.Request[ReconcilerID]{
		ID: NewReconcilerID(tenantID, "12345678"),
	}
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
			name: "TestCase1",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase2",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase3",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase5",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase6",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if got := hr.Reconcile(tt.args.ctx, tt.args.request); reflect.DeepEqual(got, tt.want) {
				t.Errorf("HostReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostReconciler_deleteHost(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Positive test Case for  deleting host nic resource of Host",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:  context.Background(),
				host: host,
			},
			wantErr: false,
		},
		{
			name: "Negative test Case for  deleting host nic resource of Host",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args:    args{ctx: context.Background()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHost(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:dupl//nolint:dupl //this is with deleteHostGpuByHost.
func TestHostReconciler_deleteHostGpuByHost(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		hostres *computev1.HostResource
	}
	host := inv_testing.CreateHost(t, nil, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Positive test case for deleting host GPU with ID",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.Background(),
				hostres: host,
			},
			wantErr: false,
		},
		{
			name: "Negative test case for deleting host GPU with ID",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostGpus: []*computev1.HostgpuResource{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHostGpuByHost(tt.args.ctx, tt.args.hostres); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHostGpuByHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:dupl //this is with deleteHostNicByHost.
func TestHostReconciler_deleteHostNicByHost(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		hostres *computev1.HostResource
	}
	host := inv_testing.CreateHost(t, nil, nil)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Positive test Case for deleting host NIC with ID",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.Background(),
				hostres: host,
			},
			wantErr: false,
		},
		{
			name: "Negative test Case for deleting host NIC with ID",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostNics: []*computev1.HostnicResource{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHostNicByHost(tt.args.ctx, tt.args.hostres); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHostNicByHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteIPsByHostNic(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		hostNic *computev1.HostnicResource
	}
	hostRes := inv_testing.CreateHost(t, nil, nil)
	hostNic := inv_testing.CreateHostNic(t, hostRes)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "positive test case for Deleting IP address with ID",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.Background(),
				hostNic: hostNic,
			},
			wantErr: false,
		},
		{
			name: "Negative test case for Deleting IP address with ID",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx: context.Background(),
				hostNic: &computev1.HostnicResource{
					ResourceId: "12345678",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteIPsByHostNic(tt.args.ctx, tt.args.hostNic); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteIPsByHostNic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:dupl //this is with deleteHostStorageByHost.
func TestHostReconciler_deleteHostStorageByHost(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	hostRes := inv_testing.CreateHost(t, nil, nil)
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		hostres *computev1.HostResource
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Positive test case for deleting host storage with ID",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.Background(),
				hostres: hostRes,
			},
			wantErr: false,
		},
		{
			name: "Negative test case for deleting host storage with ID  ",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostStorages: []*computev1.HoststorageResource{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHostStorageByHost(tt.args.ctx, tt.args.hostres); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHostStorageByHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:dupl //this is with deleteHostUsbByHost.
func TestHostReconciler_deleteHostUsbByHost(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	hostRes := inv_testing.CreateHost(t, nil, nil)
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Negative test Case for Deleting host USB with ID",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:  context.Background(),
				host: hostRes,
			},
			wantErr: false,
		},
		{
			name: "Negative test Case for Deleting host USB with ID ",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx: context.Background(),
				host: &computev1.HostResource{
					HostUsbs: []*computev1.HostusbResource{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHostUsbByHost(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHostUsbByHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_Reconcile(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ReconcilerID]
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	testRequest := rec_v2.Request[ReconcilerID]{
		ID: NewReconcilerID(tenantID, "12345678"),
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ReconcilerID]
	}{
		{
			name: "TestCase for checking reclonic resource id",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if got := hr.Reconcile(tt.args.ctx, tt.args.request); reflect.DeepEqual(got, tt.want) {
				t.Errorf("HostReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostReconciler_reconcileHost(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ReconcilerID]
		host    *computev1.HostResource
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ReconcilerID]
	}{
		{
			name: "Test Case for reclonic host with host resource values",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx:     context.Background(),
				request: rec_v2.Request[ReconcilerID]{},
				host: &computev1.HostResource{
					DesiredState: computev1.HostState_HOST_STATE_DELETED,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if got := hr.reconcileHost(tt.args.ctx, tt.args.request, tt.args.host); reflect.DeepEqual(got, tt.want) {
				t.Errorf("HostReconciler.reconcileHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostReconciler_deleteHost_Case(t *testing.T) {
	flags.FlagDisableCredentialsManagement = flag.Bool("iname", false, "")
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Negative test case for deleting host by ingroing the values of HostGpus,HostUsbs,HostStorages",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args:    args{ctx: context.Background()},
			wantErr: true,
		},
	}
	defer func() {
		flags.FlagDisableCredentialsManagement = flag.Bool("n", false, "")
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHost(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteHost_Case1(t *testing.T) {
	flags.FlagDisableCredentialsManagement = flag.Bool("jname", false, "")
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Negative test case for deleting host by ingnoring the values of HostGpus and HostUsbs ",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx: context.Background(),
				host: &computev1.HostResource{
					CurrentState: computev1.HostState_HOST_STATE_UNTRUSTED,
					HostStorages: []*computev1.HoststorageResource{
						{
							ResourceId: "host",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	defer func() {
		flags.FlagDisableCredentialsManagement = flag.Bool("j", false, "")
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHost(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteHost_Case2(t *testing.T) {
	flags.FlagDisableCredentialsManagement = flag.Bool("kname", false, "")
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Negative test case for deleting host by ingoring values of HostGpus",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx: context.Background(),
				host: &computev1.HostResource{
					CurrentState: computev1.HostState_HOST_STATE_UNTRUSTED,
					HostUsbs: []*computev1.HostusbResource{
						{
							ResourceId: "usbs",
						},
					},
					HostStorages: []*computev1.HoststorageResource{
						{
							ResourceId: "host",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	defer func() {
		flags.FlagDisableCredentialsManagement = flag.Bool("k", false, "")
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHost(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteHost_Case3(t *testing.T) {
	flags.FlagDisableCredentialsManagement = flag.Bool("lname", false, "")

	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Negative test case for deleting host",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx: context.Background(),
				host: &computev1.HostResource{
					CurrentState: computev1.HostState_HOST_STATE_UNTRUSTED,
					HostGpus: []*computev1.HostgpuResource{
						{
							ResourceId: "ups",
						},
					},
					HostUsbs: []*computev1.HostusbResource{
						{
							ResourceId: "usbs",
						},
					},
					HostStorages: []*computev1.HoststorageResource{
						{
							ResourceId: "host",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	defer func() {
		flags.FlagDisableCredentialsManagement = flag.Bool("l", false, "")
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHost(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_deleteHostNicByHost_Case(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		hostres *computev1.HostResource
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Negative Test Case for deleting host nic id",
			fields: fields{
				invClient: om_testing.InvClient,
			},
			args: args{
				ctx: context.Background(),
				hostres: &computev1.HostResource{
					HostNics: []*computev1.HostnicResource{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient: tt.fields.invClient,
			}
			if err := hr.deleteHostNicByHost(tt.args.ctx, tt.args.hostres); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.deleteHostNicByHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostReconciler_checkIfInstanceIsAssociated(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	type fields struct {
		invClient     *invclient.OnboardingInventoryClient
		enableTracing bool
	}
	type args struct {
		ctx  context.Context
		host *computev1.HostResource
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Negative Test Case for empty host resourceId",
			fields: fields{
				invClient:     om_testing.InvClient,
				enableTracing: false,
			},
			args: args{
				ctx: context.Background(),
				host: &computev1.HostResource{
					Instance: &computev1.InstanceResource{
						ResourceId: uuid.NewString(),
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hr := &HostReconciler{
				invClient:     tt.fields.invClient,
				enableTracing: tt.fields.enableTracing,
			}
			if err := hr.checkIfInstanceIsAssociated(tt.args.ctx, tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("HostReconciler.checkIfInstanceIsAssociated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
