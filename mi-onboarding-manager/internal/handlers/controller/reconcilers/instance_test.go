// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	v16 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"github.com/stretchr/testify/mock"
)

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
			name: "Positive",
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
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{}, errors.New("err"))

	mockInstance2 := &computev1.InstanceResource{
		DesiredState: computev1.InstanceState_INSTANCE_STATE_DELETED,
		CurrentState: computev1.InstanceState_INSTANCE_STATE_DELETED,
	}
	mockResource2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: mockInstance2,
		},
	}
	mockInvClient2 := &onboarding.MockInventoryClient{}
	mockInvClient2.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource2,
	}, nil)

	mockInstance3 := &computev1.InstanceResource{
		DesiredState: computev1.InstanceState_INSTANCE_STATE_INSTALLED,
	}
	mockResource3 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: mockInstance3,
		},
	}
	mockInvClient3 := &onboarding.MockInventoryClient{}
	mockInvClient3.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource3,
	}, nil)
	mockInstance4 := &computev1.InstanceResource{
		DesiredState: computev1.InstanceState_INSTANCE_STATE_DELETED,
	}
	mockResource4 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: mockInstance4,
		},
	}
	mockInvClient4 := &onboarding.MockInventoryClient{}
	mockInvClient4.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource4,
	}, nil)
	mockInvClient4.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)

	mockInvClient5 := &onboarding.MockInventoryClient{}
	mockInvClient5.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource4,
	}, nil)
	mockInvClient5.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	mockInstance7 := &computev1.InstanceResource{
		ResourceId:   "inst-084d9b08",
		DesiredState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
		Host: &computev1.HostResource{
			ResourceId: "host-084d9b01",
			Name:       "name",
			MgmtIp:     "00.00.00.00",
		},
		Os: &v16.OperatingSystemResource{
			ResourceId:  "os-093dd2d7",
			ProfileName: "profilename",
			RepoUrl:     "osUrl;overlayUrl",
		},
	}
	mockResource7 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: mockInstance7,
		},
	}
	mockHost7 := &computev1.HostResource{
		ResourceId: "host-084d9b52",
		BmcIp:      "00.00.00.00",
		HostNics: []*computev1.HostnicResource{
			{
				MacAddr: "00:00:00:00:00:00",
				Host: &computev1.HostResource{
					BmcIp: "00.00.00.00",
				},
			},
		},
	}
	mockHostResource7 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost7,
		},
	}

	mockOs7 := &v16.OperatingSystemResource{
		ResourceId:  "os-093dd2d7",
		ProfileName: "profilename",
	}
	mockOsResource7 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: mockOs7,
		},
	}
	mockInvClient7 := &onboarding.MockInventoryClient{}

	mockInvClient7.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource7,
	}, nil).Once()
	mockInvClient7.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockHostResource7,
	}, nil).Once()
	mockInvClient7.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockOsResource7,
	}, nil).Once()
	mockInvClient7.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockInvClient7.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	t.Setenv("PD_IP", "000.000.0.000")
	defer os.Unsetenv("PD_IP")
	t.Setenv("IMAGE_TYPE", "prod_focal-ms")
	defer os.Unsetenv("IMAGE_TYPE")
	dirPath, _ := os.Getwd()
	dirPath, _ = strings.CutSuffix(dirPath, "internal/handlers/controller/reconcilers")
	dirPaths := dirPath + "/cmd/onboardingmgr"
	err := os.Chdir(dirPaths)
	if err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
	}

	mockInstance8 := &computev1.InstanceResource{
		ResourceId:   "inst-084d9b08",
		DesiredState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
		Host: &computev1.HostResource{
			ResourceId: "host-084d9b02",
			Name:       "name",
			BmcIp:      "00.00.00.00",
			HostNics: []*computev1.HostnicResource{
				{
					MacAddr: "00:00:00:00:00:00",
					Host: &computev1.HostResource{
						BmcIp: "00.00.00.00",
					},
				},
			},
		},
		Os: &v16.OperatingSystemResource{
			ResourceId:  "os-093dd2d7",
			ProfileName: "profilename",
			RepoUrl:     "osUrl;overlayUrl",
		},
	}
	mockResource8 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: mockInstance8,
		},
	}
	mockHost8 := &computev1.HostResource{
		ResourceId: "host-084d9b03",
		BmcIp:      "00.00.00.00",
		HostNics: []*computev1.HostnicResource{
			{
				MacAddr: "00:00:00:00:00:00",
			},
		},
	}
	mockHostResource8 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost8,
		},
	}

	mockOs8 := &v16.OperatingSystemResource{
		ResourceId:  "os-093dd2d7",
		ProfileName: "profilename",
		RepoUrl:     "osUrl;overlayUrl",
	}
	mockOsResource8 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: mockOs8,
		},
	}
	mockInvClient8 := &onboarding.MockInventoryClient{}

	mockInvClient8.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource8,
	}, nil).Once()
	mockInvClient8.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockHostResource8,
	}, nil).Once()
	mockInvClient8.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockOsResource8,
	}, nil).Once()
	mockInvClient8.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	mockInvClient8.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, nil)
	mockInstance10 := &computev1.InstanceResource{
		ResourceId:   "inst-084d9b08",
		DesiredState: computev1.InstanceState_INSTANCE_STATE_RUNNING,
		Host: &computev1.HostResource{
			ResourceId: "host-084d9b06",
			Name:       "name",
			MgmtIp:     "00.00.00.00",
			BmcIp:      "00.00.00.00",
			HostNics: []*computev1.HostnicResource{
				{
					MacAddr: "00:00:00:00:00:00",
				},
			},
		},
		Os: &v16.OperatingSystemResource{
			ResourceId:  "os-093dd2d7",
			ProfileName: "profilename",
			RepoUrl:     "osUrl;overlayUrl",
		},
	}
	mockResource10 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Instance{
			Instance: mockInstance10,
		},
	}
	mockHost10 := &computev1.HostResource{
		ResourceId: "host-084d9b04",
		BmcIp:      "00.00.00.00",
		HostNics: []*computev1.HostnicResource{
			{
				MacAddr: "00:00:00:00:00:00",
			},
		},
	}
	mockHostResource10 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Host{
			Host: mockHost10,
		},
	}

	mockOs10 := &v16.OperatingSystemResource{
		ResourceId:  "os-093dd2d7",
		ProfileName: "profilename",
		RepoUrl:     "osUrl;overlayUrl",
	}
	mockOsResource10 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Os{
			Os: mockOs10,
		},
	}
	mockInvClient10 := &onboarding.MockInventoryClient{}

	mockInvClient10.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockResource10,
	}, nil).Once()
	mockInvClient10.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockHostResource10,
	}, nil).Once()
	mockInvClient10.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: mockOsResource10,
	}, nil).Once()
	mockInvClient10.On("Update", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(&inv_v1.UpdateResourceResponse{}, errors.New("err"))
	mockInvClient10.On("List", mock.Anything, mock.Anything).Return(&inv_v1.ListResourcesResponse{}, nil)
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ResourceID]
	}{
		{
			name: "TestCase4",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient4,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase5",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient5,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase7",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient7,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase8",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient8,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase10",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient10,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: rec_v2.Request[ResourceID]{},
			},
			want: testRequest.Ack(),
		},
	}
	originalDir, _ := os.Getwd()
	err = os.Chdir(originalDir)
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
