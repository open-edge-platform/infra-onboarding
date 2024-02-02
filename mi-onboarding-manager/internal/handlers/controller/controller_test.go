// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package controller

import (
	"sync"
	"testing"

	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/handlers/controller/reconcilers"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
)

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
			_, err := New(tt.args.invClient)
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
		invClient   *invclient.OnboardingInventoryClient
		filters     map[inv_v1.ResourceKind]Filter
		controllers map[inv_v1.ResourceKind]*rec_v2.Controller[reconcilers.ResourceID]
		wg          *sync.WaitGroup
		stop        chan bool
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

