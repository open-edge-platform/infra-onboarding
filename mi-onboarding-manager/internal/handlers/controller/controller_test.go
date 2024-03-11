// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package controller

import (
	"context"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/testing"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/handlers/controller/reconcilers"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	policyPath := projectRoot + "/build"
	migrationsDir := projectRoot + "/build"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestReconcileEvent(t *testing.T) {
	// increase default reconciliation interval
	defaultTickerPeriod = 30 * time.Second
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	nbHandler, err := New(om_testing.InvClient, false)
	require.NoError(t, err)

	// Use a mock reconciler
	doneHost := make(chan bool, 1)
	controllerHost := rec_v2.NewController[reconcilers.ResourceID](func(ctx context.Context,
		request rec_v2.Request[reconcilers.ResourceID],
	) rec_v2.Directive[reconcilers.ResourceID] {
		doneHost <- true
		return request.Ack()
	}, rec_v2.WithParallelism(1))
	nbHandler.controllers[inv_v1.ResourceKind_RESOURCE_KIND_HOST] = controllerHost

	doneOS := make(chan bool, 1)
	controllerOS := rec_v2.NewController[reconcilers.ResourceID](func(ctx context.Context,
		request rec_v2.Request[reconcilers.ResourceID],
	) rec_v2.Directive[reconcilers.ResourceID] {
		doneOS <- true
		return request.Ack()
	}, rec_v2.WithParallelism(1))
	nbHandler.controllers[inv_v1.ResourceKind_RESOURCE_KIND_OS] = controllerOS

	doneInstance := make(chan bool, 1)
	controllerInstance := rec_v2.NewController[reconcilers.ResourceID](func(ctx context.Context,
		request rec_v2.Request[reconcilers.ResourceID],
	) rec_v2.Directive[reconcilers.ResourceID] {
		doneInstance <- true
		return request.Ack()
	}, rec_v2.WithParallelism(1))
	nbHandler.controllers[inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE] = controllerInstance

	err = nbHandler.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		nbHandler.Stop()
	})

	host := inv_testing.CreateHostNoCleanup(t, nil, nil, nil, nil)
	osRes := inv_testing.CreateOsNoCleanup(t)
	inst := inv_testing.CreateInstanceNoCleanup(t, host, osRes)

	assert.True(t, <-doneOS)
	assert.True(t, <-doneHost)
	assert.True(t, <-doneInstance)

	// Do hard delete directly, the reconciler is fake and won't actually delete the resource
	inv_testing.HardDeleteInstance(t, inst.ResourceId)
	inv_testing.HardDeleteHost(t, host.ResourceId)
	inv_testing.DeleteResource(t, osRes.ResourceId)

	// UPDATED event for Host and Instance
	assert.True(t, <-doneHost)
	assert.True(t, <-doneInstance)

	// DELETED event for OS and Instance
	assert.True(t, <-doneOS)
	assert.True(t, <-doneInstance)

	select {
	case v := <-doneHost:
		t.Errorf("Unexpected Host message received on channel: %v", v)
		t.Fail()
	case v := <-doneInstance:
		t.Errorf("Unexpected Instance message received on channel: %v", v)
		t.Fail()
	case v := <-doneOS:
		t.Errorf("Unexpected OS message received on channel: %v", v)
		t.Fail()
	case <-time.After(3 * time.Second):
		break
	}
}

func TestReconcileAll(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	nbHandler, err := New(om_testing.InvClient, false)
	require.NoError(t, err)

	// Use a mock reconciler
	doneHost := make(chan bool, 1)
	controllerHost := rec_v2.NewController[reconcilers.ResourceID](func(ctx context.Context,
		request rec_v2.Request[reconcilers.ResourceID],
	) rec_v2.Directive[reconcilers.ResourceID] {
		doneHost <- true
		return request.Ack()
	}, rec_v2.WithParallelism(1))
	nbHandler.controllers[inv_v1.ResourceKind_RESOURCE_KIND_HOST] = controllerHost

	doneInstance := make(chan bool, 1)
	controllerInstance := rec_v2.NewController[reconcilers.ResourceID](func(ctx context.Context,
		request rec_v2.Request[reconcilers.ResourceID],
	) rec_v2.Directive[reconcilers.ResourceID] {
		doneInstance <- true
		return request.Ack()
	}, rec_v2.WithParallelism(1))
	nbHandler.controllers[inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE] = controllerInstance

	doneOS := make(chan bool, 1)
	controllerOS := rec_v2.NewController[reconcilers.ResourceID](func(ctx context.Context,
		request rec_v2.Request[reconcilers.ResourceID],
	) rec_v2.Directive[reconcilers.ResourceID] {
		doneOS <- true
		return request.Ack()
	}, rec_v2.WithParallelism(1))
	nbHandler.controllers[inv_v1.ResourceKind_RESOURCE_KIND_OS] = controllerOS

	// Create beforehand the resources
	host := inv_testing.CreateHostNoCleanup(t, nil, nil, nil, nil)
	osRes := inv_testing.CreateOsNoCleanup(t)
	inst := inv_testing.CreateInstanceNoCleanup(t, host, osRes)

	// Rewrite the ticker period
	defaultTickerPeriod = 2 * time.Second

	err = nbHandler.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		nbHandler.Stop()
	})

	// Initial reconcileAll
	time.Sleep(1 * time.Second)
	assert.True(t, <-doneHost)
	assert.True(t, <-doneInstance)
	assert.True(t, <-doneOS)

	// delayed CREATED events
	assert.True(t, <-doneHost)
	assert.True(t, <-doneInstance)
	assert.True(t, <-doneOS)

	// Do hard delete directly, the reconciler is fake and won't actually delete the resource
	inv_testing.HardDeleteInstance(t, inst.ResourceId)
	inv_testing.HardDeleteHost(t, host.ResourceId)
	inv_testing.DeleteResource(t, osRes.ResourceId)

	// UPDATED event for Host and Instance
	assert.True(t, <-doneHost)
	assert.True(t, <-doneInstance)

	// DELETED event for OS and Instance
	assert.True(t, <-doneOS)
	assert.True(t, <-doneInstance)

	select {
	case v := <-doneHost:
		t.Errorf("Unexpected Host message received on channel: %v", v)
		t.Fail()
	case v := <-doneInstance:
		t.Errorf("Unexpected Instance message received on channel: %v", v)
		t.Fail()
	case v := <-doneOS:
		t.Errorf("Unexpected OS message received on channel: %v", v)
		t.Fail()
	case <-time.After(3 * time.Second):
		break
	}
}

func TestReconcileNoControllers(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	nbHandler, err := New(om_testing.InvClient, false)
	require.NoError(t, err)

	// Use a mock reconciler
	doneHost := make(chan bool, 1)
	controllerHost := rec_v2.NewController[reconcilers.ResourceID](func(ctx context.Context,
		request rec_v2.Request[reconcilers.ResourceID],
	) rec_v2.Directive[reconcilers.ResourceID] {
		doneHost <- true
		return request.Ack()
	}, rec_v2.WithParallelism(1))
	nbHandler.controllers[inv_v1.ResourceKind_RESOURCE_KIND_HOST] = controllerHost

	doneInstance := make(chan bool, 1)
	controllerInstance := rec_v2.NewController[reconcilers.ResourceID](func(ctx context.Context,
		request rec_v2.Request[reconcilers.ResourceID],
	) rec_v2.Directive[reconcilers.ResourceID] {
		doneInstance <- true
		return request.Ack()
	}, rec_v2.WithParallelism(1))
	nbHandler.controllers[inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE] = controllerInstance

	doneOS := make(chan bool, 1)
	controllerOS := rec_v2.NewController[reconcilers.ResourceID](func(ctx context.Context,
		request rec_v2.Request[reconcilers.ResourceID],
	) rec_v2.Directive[reconcilers.ResourceID] {
		doneOS <- true
		return request.Ack()
	}, rec_v2.WithParallelism(1))
	nbHandler.controllers[inv_v1.ResourceKind_RESOURCE_KIND_OS] = controllerOS

	err = nbHandler.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		nbHandler.Stop()
	})

	delete(nbHandler.controllers, inv_v1.ResourceKind_RESOURCE_KIND_HOST)
	delete(nbHandler.controllers, inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE)
	delete(nbHandler.controllers, inv_v1.ResourceKind_RESOURCE_KIND_OS)
	newHost := inv_testing.CreateHost(t, nil, nil, nil, nil)
	newOs := inv_testing.CreateOs(t)
	inv_testing.CreateInstance(t, newHost, newOs)

	select {
	case v := <-doneHost:
		t.Errorf("Unexpected Host message received on channel: %v", v)
		t.Fail()
	case v := <-doneInstance:
		t.Errorf("Unexpected Instance message received on channel: %v", v)
		t.Fail()
	case v := <-doneOS:
		t.Errorf("Unexpected OS message received on channel: %v", v)
		t.Fail()
	case <-time.After(3 * time.Second):
		break
	}
}

func TestFilterEventErrors(t *testing.T) {
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})

	nbHandler, err := New(om_testing.InvClient, false)
	require.NoError(t, err)

	t.Run("FailedToValidateEvent", func(t *testing.T) {
		result := nbHandler.filterEvent(&inv_v1.SubscribeEventsResponse{
			ClientUuid: "invalid uuid",
			ResourceId: "os-12345678",
			Resource:   &inv_v1.Resource{Resource: &inv_v1.Resource_Os{}},
			EventKind:  inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
		})
		require.False(t, result)
	})

	t.Run("FailedUnexpectedResource", func(t *testing.T) {
		result := nbHandler.filterEvent(&inv_v1.SubscribeEventsResponse{
			ClientUuid: "",
			ResourceId: "xyz-12345678",
			Resource:   &inv_v1.Resource{Resource: &inv_v1.Resource_Os{}},
			EventKind:  inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
		})
		require.False(t, result)
	})

	t.Run("FailedNoFilterForExpectedKind", func(t *testing.T) {
		delete(nbHandler.filters, inv_v1.ResourceKind_RESOURCE_KIND_OS)
		result := nbHandler.filterEvent(&inv_v1.SubscribeEventsResponse{
			ClientUuid: "",
			ResourceId: "os-12345678",
			Resource:   &inv_v1.Resource{Resource: &inv_v1.Resource_Os{}},
			EventKind:  inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
		})
		// all events are accepted if no filter
		require.True(t, result)
	})
}

func TestNew(t *testing.T) {
	type args struct {
		invClient *invclient.OnboardingInventoryClient
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				invClient: &invclient.OnboardingInventoryClient{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.args.invClient, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestOnboardingController_Start(t *testing.T) {
	invClient := &onboarding.MockInventoryClient{}
	invClient.On("FindAll", mock.Anything, mock.Anything, mock.Anything).
		Return([]string{}, nil)
	invClient1 := &onboarding.MockInventoryClient{}
	invClient1.On("FindAll", mock.Anything, mock.Anything, mock.Anything).
		Return([]string{}, errors.New("Error"))
	invClient2 := &onboarding.MockInventoryClient{}
	invClient2.On("FindAll", mock.Anything, mock.Anything, mock.Anything).
		Return([]string{"64-567"}, nil)
	invClient3 := &onboarding.MockInventoryClient{}
	invClient3.On("FindAll", mock.Anything, mock.Anything, mock.Anything).
		Return([]string{"os-567"}, nil)
	type fields struct {
		invClient   *invclient.OnboardingInventoryClient
		filters     map[inv_v1.ResourceKind]Filter
		controllers map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID]
		wg          *sync.WaitGroup
		stop        chan bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: invClient},

				filters:     make(map[inv_v1.ResourceKind]Filter),
				controllers: make(map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID]),
				wg:          &sync.WaitGroup{},
				stop:        make(chan bool),
			},
			wantErr: false,
		},
		{
			name: "Negative",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: invClient1},

				filters:     make(map[inv_v1.ResourceKind]Filter),
				controllers: make(map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID]),
				wg:          &sync.WaitGroup{},
				stop:        make(chan bool),
			},
			wantErr: true,
		},
		{
			name: "Negative1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: invClient2},

				filters:     make(map[inv_v1.ResourceKind]Filter),
				controllers: make(map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID]),
				wg:          &sync.WaitGroup{},
				stop:        make(chan bool),
			},
			wantErr: true,
		},
		{
			name: "Negative2",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{Client: invClient3},

				filters:     make(map[inv_v1.ResourceKind]Filter),
				controllers: make(map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID]),
				wg:          &sync.WaitGroup{},
				stop:        make(chan bool),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obc := &OnboardingController{
				invClient:   tt.fields.invClient,
				filters:     tt.fields.filters,
				controllers: tt.fields.controllers,
				wg:          tt.fields.wg,
				stop:        tt.fields.stop,
			}
			go func() {
				obc.invClient.Watcher <- &inv_client.WatchEvents{
					Event: &inv_v1.SubscribeEventsResponse{
						ResourceId: "64-567",
					},
				}
			}()
			go func() {
				obc.invClient.Watcher <- &inv_client.WatchEvents{
					Event: &inv_v1.SubscribeEventsResponse{
						ResourceId: "os-567",
					},
				}
			}()
			if err := obc.Start(); (err != nil) != tt.wantErr {
				t.Errorf("OnboardingController.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOnboardingController_Stop(t *testing.T) {
	type fields struct {
		invClient   *invclient.OnboardingInventoryClient
		filters     map[inv_v1.ResourceKind]Filter
		controllers map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID]
		wg          *sync.WaitGroup
		stop        chan bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Positive",
			fields: fields{
				invClient:   nil,
				filters:     make(map[inv_v1.ResourceKind]Filter),
				controllers: make(map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID]),
				wg:          &sync.WaitGroup{},
				stop:        make(chan bool),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obc := &OnboardingController{
				invClient:   tt.fields.invClient,
				filters:     tt.fields.filters,
				controllers: tt.fields.controllers,
				wg:          tt.fields.wg,
				stop:        tt.fields.stop,
			}
			obc.Stop()
		})
	}
}

func Test_instanceEventFilter(t *testing.T) {
	type args struct {
		event *inv_v1.SubscribeEventsResponse
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test Updated Event",
			args: args{&inv_v1.SubscribeEventsResponse{
				EventKind: inv_v1.SubscribeEventsResponse_EVENT_KIND_UPDATED,
			}},
			want: true,
		},
		{
			name: "Test Create Event",
			args: args{&inv_v1.SubscribeEventsResponse{
				EventKind: inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED,
			}},
			want: true,
		},
		{
			name: "Test Delete Event",
			args: args{&inv_v1.SubscribeEventsResponse{
				EventKind: inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
			}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := instanceEventFilter(tt.args.event); got != tt.want {
				t.Errorf("instanceEventFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hostEventFilter(t *testing.T) {
	type args struct {
		event *inv_v1.SubscribeEventsResponse
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test Delete Event",
			args: args{&inv_v1.SubscribeEventsResponse{
				EventKind: inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
			}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostEventFilter(tt.args.event); got != tt.want {
				t.Errorf("hostEventFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_osEventFilter(t *testing.T) {
	type args struct {
		event *inv_v1.SubscribeEventsResponse
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test Updated Event",
			args: args{&inv_v1.SubscribeEventsResponse{
				EventKind: inv_v1.SubscribeEventsResponse_EVENT_KIND_UPDATED,
			}},
			want: true,
		},
		{
			name: "Test Create Event",
			args: args{&inv_v1.SubscribeEventsResponse{
				EventKind: inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED,
			}},
			want: true,
		},
		{
			name: "Test Delete Event",
			args: args{&inv_v1.SubscribeEventsResponse{
				EventKind: inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
			}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := osEventFilter(tt.args.event); got != tt.want {
				t.Errorf("osEventFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOnboardingController_filterEvent(t *testing.T) {
	type fields struct {
		filters map[inv_v1.ResourceKind]Filter
	}
	type args struct {
		event *inv_v1.SubscribeEventsResponse
	}
	mockEventValid := &inv_v1.SubscribeEventsResponse{
		ResourceId: "valid_resource_id",
		EventKind:  inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED,
	}
	mockEventInvalid := &inv_v1.SubscribeEventsResponse{
		ResourceId: "invalid_resource_id",
		EventKind:  inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED,
	}
	mockEventInvalids := &inv_v1.SubscribeEventsResponse{
		// ResourceId: "invalid_resource_id",
		EventKind: inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED,
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TestCase1",
			fields: fields{
				filters: map[inv_v1.ResourceKind]Filter{
					inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE: func(event *inv_v1.SubscribeEventsResponse) bool {
						return true
					},
				},
			},
			args: args{
				event: mockEventValid,
			},
			want: false,
		},
		{
			name: "TestCase2",
			fields: fields{
				filters: map[inv_v1.ResourceKind]Filter{
					inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE: func(event *inv_v1.SubscribeEventsResponse) bool {
						return false
					},
				},
			},
			args: args{
				event: mockEventInvalid,
			},
			want: false,
		},
		{
			name: "TestCase3",
			fields: fields{
				filters: map[inv_v1.ResourceKind]Filter{},
			},
			args: args{
				event: mockEventInvalid,
			},
			want: false,
		},
		{
			name: "TestCase4",
			fields: fields{
				filters: map[inv_v1.ResourceKind]Filter{},
			},
			args: args{
				event: mockEventInvalids,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obc := &OnboardingController{
				filters: tt.fields.filters,
			}
			if got := obc.filterEvent(tt.args.event); got != tt.want {
				t.Errorf("OnboardingController.filterEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}
