// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	dkam "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	onboarding "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onboarding"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestNewOsReconciler(t *testing.T) {
	type args struct {
		c *invclient.OnboardingInventoryClient
	}
	tests := []struct {
		name string
		args args
		want *OsReconciler
	}{
		{
			name: "Positive",
			args: args{
				c: &invclient.OnboardingInventoryClient{},
			},
			want: &OsReconciler{
				invClient: &invclient.OnboardingInventoryClient{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewOsReconciler(tt.args.c, false); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOsReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOsReconciler_Reconcile(t *testing.T) {
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ResourceID]
	}
	testRequest := rec_v2.Request[ResourceID]{
		ID: ResourceID("test-id"),
	}
	mockInvClient := &onboarding.MockInventoryClient{}
	mockInvClient.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{}, errors.New("err"))
	t.Setenv("DISABLE_FEATUREX", "true")
	mockInvClient1 := &onboarding.MockInventoryClient{}
	mockInvClient1.On("Get", mock.Anything, mock.Anything).Return(&inv_v1.GetResourceResponse{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Os{
				Os: &osv1.OperatingSystemResource{
					ResourceId: "os-084d9b08",
				},
			},
		},
	}, nil)

	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ResourceID]
	}{
		{
			name: "TestCase",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "TestCase1",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{
					Client: mockInvClient1,
				},
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
	}
	defer func() {
		os.Unsetenv("DISABLE_FEATUREX")
		os.Unsetenv("DKAMHOST")
		os.Unsetenv("DKAMPORT")
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osr := &OsReconciler{
				invClient: tt.fields.invClient,
			}
			if got := osr.Reconcile(tt.args.ctx, tt.args.request); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OsReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSameOSResource(t *testing.T) {
	type args struct {
		originalOSRes *osv1.OperatingSystemResource
		updatedOSRes  *osv1.OperatingSystemResource
		fieldmask     *fieldmaskpb.FieldMask
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "Test Case 1",
			args:    args{},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsSameOSResource(tt.args.originalOSRes, tt.args.updatedOSRes, tt.args.fieldmask)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsSameOSResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsSameOSResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPopulateOSResourceFromDKAMResponse(t *testing.T) {
	type args struct {
		dkamResponse *dkam.GetArtifactsResponse
	}
	tests := []struct {
		name    string
		args    args
		want    *osv1.OperatingSystemResource
		want1   *fieldmaskpb.FieldMask
		wantErr bool
	}{
		{
			name:    "Test case 1",
			args:    args{},
			want:    nil,
			want1:   nil,
			wantErr: true,
		},
		{
			name: "Test case 2",
			args: args{
				dkamResponse: &dkam.GetArtifactsResponse{OverlayscriptUrl: "url"},
			},
			want:    &osv1.OperatingSystemResource{RepoUrl: ";url"},
			want1:   &fieldmaskpb.FieldMask{Paths: []string{"repo_url"}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := PopulateOSResourceFromDKAMResponse(tt.args.dkamResponse)
			if (err != nil) != tt.wantErr {
				t.Errorf("PopulateOSResourceFromDKAMResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PopulateOSResourceFromDKAMResponse() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("PopulateOSResourceFromDKAMResponse() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
