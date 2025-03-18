// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	dkam_testing "github.com/open-edge-platform/infra-onboarding/dkam/testing"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	om_testing "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/testing"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell"
)

func TestNewTemplate(t *testing.T) {
	type args struct {
		tpData string
		name   string
		ns     string
	}
	tests := []struct {
		name string
		args args
		want *tink.Template
	}{
		{
			name: "Test case 1",
			args: args{
				tpData: "templateData",
				name:   "exampleTemplate",
				ns:     "exampleNamespace",
			},
			want: &tink.Template{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Template",
					APIVersion: "tinkerbell.org/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exampleTemplate",
					Namespace: "exampleNamespace",
				},
				Spec: tink.TemplateSpec{
					Data: func() *string {
						s := "templateData"
						return &s
					}(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tinkerbell.NewTemplate(tt.args.tpData, tt.args.name, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateTemplateForProd(t *testing.T) {
	dkam_testing.PrepareTestCaCertificateFile(t)
	dkam_testing.PrepareTestInfraConfig(t)
	type args struct {
		k8sNamespace string
		deviceInfo   onboarding_types.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    *tink.Template
		wantErr bool
	}{
		{
			name:    "Test Case",
			args:    args{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test Case1",
			args: args{
				deviceInfo: onboarding_types.DeviceInfo{
					OsType:           osv1.OsType_OS_TYPE_MUTABLE,
					TenantID:         "test-tenantid",
					Hostname:         "test-hostname",
					AuthClientID:     "test-client-id",
					AuthClientSecret: "test-client-secret",
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tinkerbell.GenerateTemplateForProd(tt.args.k8sNamespace, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTemplateForProd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestCreateTemplateIfNotExists(t *testing.T) {
	type args struct {
		ctx      context.Context
		k8sCli   client.Client
		template *tink.Template
	}
	mockClient := om_testing.MockK8sClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := om_testing.MockK8sClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient,
				template: &tink.Template{},
			},
		},
		{
			name: "Test Case1",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient1,
				template: &tink.Template{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tinkerbell.CreateTemplateIfNotExists(tt.args.ctx, tt.args.k8sCli,
				tt.args.template); (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplateIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:dupl //this is with tink.Template as args.
func TestCreateTemplateIfNotExists_Case(t *testing.T) {
	type args struct {
		ctx      context.Context
		k8sCli   client.Client
		template *tink.Template
	}
	mockClient := om_testing.MockK8sClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := om_testing.MockK8sClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	mockClient2 := om_testing.MockK8sClient{}
	mockClient2.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(k8sErrors.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	mockClient2.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient3 := om_testing.MockK8sClient{}
	mockClient3.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(k8sErrors.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	mockClient3.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient,
				template: &tink.Template{},
			},
		},
		{
			name: "Test Case1",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient1,
				template: &tink.Template{},
			},
			wantErr: true,
		},
		{
			name: "Test Case2",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient2,
				template: &tink.Template{},
			},
			wantErr: false,
		},
		{
			name: "Test Case3",
			args: args{
				ctx:      context.Background(),
				k8sCli:   &mockClient3,
				template: &tink.Template{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tinkerbell.CreateTemplateIfNotExists(tt.args.ctx, tt.args.k8sCli,
				tt.args.template); (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplateIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
