// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	rec_v2 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.lib-go/pkg/controller/v2"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/handlers/controller/reconcilers"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/invclient"
	dkam_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/testing"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/inventory/v1"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	run := m.Run()
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestNew(t *testing.T) {
	dkam_testing.CreateInventoryDKAMClientForTesting()
	t.Cleanup(func() {
		dkam_testing.DeleteInventoryDKAMClientForTesting()
	})
	nbHandler, err := New(dkam_testing.InvClient, false)
	require.NoError(t, err)
	err = nbHandler.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		nbHandler.Stop()
	})
	type args struct {
		invClient     *invclient.DKAMInventoryClient
		enableTracing bool
	}
	tests := []struct {
		name    string
		args    args
		want    *DKAMController
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				invClient:     dkam_testing.InvClient,
				enableTracing: false,
			},
			want:    &DKAMController{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.invClient, tt.args.enableTracing)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDKAMController_filterEvent(t *testing.T) {
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
		EventKind: inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED,
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Test dkam controller -filter event with valid filter",
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
			name: "Test dkam controller -filter event with invalid filter",
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
			name: "Test dkamController -filter event with no filters",
			fields: fields{
				filters: map[inv_v1.ResourceKind]Filter{},
			},
			args: args{
				event: mockEventInvalid,
			},
			want: false,
		},
		{
			name: "Test dkamController -Filter event with no ResourceId",
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
			obc := &DKAMController{
				filters: tt.fields.filters,
			}
			if got := obc.filterEvent(tt.args.event); got != tt.want {
				t.Errorf("DKAMController.filterEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDKAMController_filterEvent_Case(t *testing.T) {
	type fields struct {
		filters map[inv_v1.ResourceKind]Filter
	}
	type args struct {
		event *inv_v1.SubscribeEventsResponse
	}
	mockEventValid := &inv_v1.SubscribeEventsResponse{
		ClientUuid: "valid_resource_id",
		EventKind:  inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED,
	}
	mockEventInvalid := &inv_v1.SubscribeEventsResponse{
		ResourceId: "host-084d9b08",
		EventKind:  inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED,
	}
	mockEventInvalids := &inv_v1.SubscribeEventsResponse{
		ResourceId: "inst-084d9b08",
		EventKind:  inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED,
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Test dkamController -Filter event with valid filter",
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
			name: "Test dkamController -Filter event with invalid filter",
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
			name: "Test dkamController -Filter event with no matching filter",
			fields: fields{
				filters: map[inv_v1.ResourceKind]Filter{
					inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE: func(event *inv_v1.SubscribeEventsResponse) bool {
						return false
					},
				},
			},
			args: args{
				event: mockEventInvalids,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obc := &DKAMController{
				filters: tt.fields.filters,
			}
			obc.filterEvent(tt.args.event)
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

func TestDKAMController_reconcileResource(t *testing.T) {
	dkam_testing.CreateInventoryDKAMClientForTesting()
	t.Cleanup(func() {
		dkam_testing.DeleteInventoryDKAMClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil, nil, nil)
	type fields struct {
		filters map[inv_v1.ResourceKind]Filter
	}
	type args struct {
		resourceID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			fields: fields{
				filters: map[inv_v1.ResourceKind]Filter{
					inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE: func(event *inv_v1.SubscribeEventsResponse) bool {
						return false
					},
				},
			},
			args: args{
				resourceID: "",
			},
			wantErr: true,
		},
		{
			name: "Test Case",
			fields: fields{
				filters: map[inv_v1.ResourceKind]Filter{
					inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE: func(event *inv_v1.SubscribeEventsResponse) bool {
						return false
					},
				},
			},
			args: args{
				resourceID: host.ResourceId,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obc := &DKAMController{
				filters: tt.fields.filters,
			}
			if err := obc.reconcileResource(tt.args.resourceID); (err != nil) != tt.wantErr {
				t.Errorf("DKAMController.reconcileResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDKAMController_reconcileAll(t *testing.T) {
	dkam_testing.CreateInventoryDKAMClientForTesting()
	t.Cleanup(func() {
		dkam_testing.DeleteInventoryDKAMClientForTesting()
	})
	type fields struct {
		invClient *invclient.DKAMInventoryClient
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "reconcile test case",
			fields: fields{
				invClient: dkam_testing.InvClient,
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obc := &DKAMController{
				invClient: tt.fields.invClient,
			}
			if err := obc.reconcileAll(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("DKAMController.reconcileAll() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconcileEvent(t *testing.T) {
	defaultTickerPeriod = 30 * time.Second
	dkam_testing.CreateInventoryDKAMClientForTesting()
	t.Cleanup(func() {
		dkam_testing.DeleteInventoryDKAMClientForTesting()
	})
	nbHandler, err := New(dkam_testing.InvClient, false)
	require.NoError(t, err)
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
	osRes := inv_testing.CreateOsNoCleanup(t)
	assert.True(t, <-doneOS)
	inv_testing.DeleteResource(t, osRes.ResourceId)
	assert.True(t, <-doneOS)
	select {
	case v := <-doneOS:
		fmt.Printf("Unexpected OS message received on channel: %v", v)
	case <-time.After(3 * time.Second):
		break
	}
}
