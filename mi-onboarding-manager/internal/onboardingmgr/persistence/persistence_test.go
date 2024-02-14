/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package persistence

import (
	"reflect"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/config"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/api"
)

func TestInitDB(t *testing.T) {
	type args struct {
		conf *config.Config
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test Case 1",
			args: args{
				&config.Config{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitDB(tt.args.conf)
		})
	}
}

func Test_runInKubernetes(t *testing.T) {
	type args struct {
		conf *config.Config
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test Case 1",
			args: args{
				conf: &config.Config{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInKubernetes(tt.args.conf)
		})
	}
}

func Test_runInDockerContainer(t *testing.T) {
	type args struct {
		conf *config.Config
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test Case 1",
			args: args{
				conf: &config.Config{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInDockerContainer(tt.args.conf)
		})
	}
}

func TestGetNodeRepository(t *testing.T) {
	tests := []struct {
		name string
		want Repository
	}{
		{
			name: "Test Case 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNodeRepository(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNodeRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetArtifactRepository(t *testing.T) {
	tests := []struct {
		name string
		want Repository
	}{
		{
			name: "Test Case 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetArtifactRepository(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetArtifactRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalToStr(t *testing.T) {
	type args struct {
		data interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Test Case 2",
			args:    args{data: nil},
			want:    "",
			wantErr: false,
		},
		{
			name:    "Test Case 3",
			args:    args{data: map[string]interface{}{"key": "value"}},
			want:    `{"key":"value"}` + "\n",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalToStr(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalToStr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MarshalToStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalStrArray(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "Test Case 1",
			args:    args{},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Test Case 2",
			args: args{
				data: "123",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalStrArray(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalStrArray() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalStrArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalOnboardingParams(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.OnboardingParams
		wantErr bool
	}{
		{
			name:    "Test Case 1",
			args:    args{},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "Test Case 2",
			args:    args{data: "123"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalOnboardingParams(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalOnboardingParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalOnboardingParams() = %v, want %v", got, tt.want)
			}
		})
	}
}
