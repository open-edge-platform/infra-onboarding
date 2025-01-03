// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	error "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
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
			if got := NewTemplate(tt.args.tpData, tt.args.name, tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateTemplateForProd(t *testing.T) {
	type args struct {
		k8sNamespace string
		deviceInfo   utils.DeviceInfo
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
			wantErr: false,
		},
		{
			name: "Test Case1",
			args: args{
				deviceInfo: utils.DeviceInfo{
					ImgType: utils.ImgTypeBkc,
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test Case2",
			args: args{
				deviceInfo: utils.DeviceInfo{
					ImgType: utils.ImgTypeJammy,
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test Case3",
			args: args{
				deviceInfo: utils.DeviceInfo{
					ImgType: utils.ImgTypeFocalMs,
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateTemplateForProd(tt.args.k8sNamespace, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTemplateForProd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateTemplateForProd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateTemplateForDI(t *testing.T) {
	type args struct {
		k8sNamespace string
		deviceInfo   utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    *tink.Template
		wantErr bool
	}{
		{
			name:    "Test Case",
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateTemplateForDI(tt.args.k8sNamespace, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTemplateForDI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateTemplateForDI() = %v, want %v", got, tt.want)
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
	mockClient := MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := MockClient{}
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
				k8sCli:   mockClient,
				template: &tink.Template{},
			},
		},
		{
			name: "Test Case1",
			args: args{
				ctx:      context.Background(),
				k8sCli:   mockClient1,
				template: &tink.Template{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateTemplateIfNotExists(tt.args.ctx, tt.args.k8sCli, tt.args.template); (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplateIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateTemplateForNodeReboot(t *testing.T) {
	type args struct {
		k8sNamespace string
		deviceInfo   utils.DeviceInfo
	}
	tests := []struct {
		name    string
		args    args
		want    *tink.Template
		wantErr bool
	}{
		{
			name:    "Test Case",
			want:    &tink.Template{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateTemplateForNodeReboot(tt.args.k8sNamespace, tt.args.deviceInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTemplateForNodeReboot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateTemplateForNodeReboot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateTemplateIfNotExists_Case(t *testing.T) {
	type args struct {
		ctx      context.Context
		k8sCli   client.Client
		template *tink.Template
	}
	mockClient := MockClient{}
	mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient1 := MockClient{}
	mockClient1.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("err"))
	mockClient2 := MockClient{}
	mockClient2.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(error.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
	mockClient2.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockClient3 := MockClient{}
	mockClient3.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(error.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "myresource"}, "resource-name"))
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
				k8sCli:   mockClient,
				template: &tink.Template{},
			},
		},
		{
			name: "Test Case1",
			args: args{
				ctx:      context.Background(),
				k8sCli:   mockClient1,
				template: &tink.Template{},
			},
			wantErr: true,
		},
		{
			name: "Test Case2",
			args: args{
				ctx:      context.Background(),
				k8sCli:   mockClient2,
				template: &tink.Template{},
			},
			wantErr: false,
		},
		{
			name: "Test Case3",
			args: args{
				ctx:      context.Background(),
				k8sCli:   mockClient3,
				template: &tink.Template{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateTemplateIfNotExists(tt.args.ctx, tt.args.k8sCli, tt.args.template); (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplateIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
