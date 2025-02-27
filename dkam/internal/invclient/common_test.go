// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
//
//nolint:testpackage // Keeping the test in the same package due to dependencies on unexported fields.
package invclient

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"

	computev1 "github.com/intel/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/intel/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/intel/infra-core/inventory/v2/pkg/api/network/v1"
	osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"
	provider_v1 "github.com/intel/infra-core/inventory/v2/pkg/api/provider/v1"
)

//nolint:funlen // it has required test cases.
func Test_getInventoryResourceAndID(t *testing.T) {
	type args struct {
		resource proto.Message
	}
	tests := []struct {
		name    string
		args    args
		want    *inv_v1.Resource
		want1   string
		wantErr bool
	}{
		{
			name:    "Empty Resource",
			args:    args{},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: true,
		},
		{
			name: "Host Resource Test",
			args: args{
				resource: proto.Clone(&computev1.HostResource{}),
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Host Storage Resource Test",
			args: args{
				resource: proto.Clone(&computev1.HoststorageResource{}),
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Host USB Resource Test",
			args: args{
				resource: proto.Clone(&computev1.HostusbResource{}),
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Host GPU Resource Test",
			args: args{
				resource: proto.Clone(&computev1.HostgpuResource{}),
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Network Resource Test",
			args: args{
				resource: proto.Clone(&network_v1.IPAddressResource{}),
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Operating System Resource Test",
			args: args{
				resource: proto.Clone(&osv1.OperatingSystemResource{}),
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "hostNic Resource Test",
			args: args{
				resource: proto.Clone(&computev1.HostnicResource{}),
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "instance Resource Test",
			args: args{
				resource: proto.Clone(&computev1.InstanceResource{}),
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "provider Resource Test",
			args: args{
				resource: proto.Clone(&provider_v1.ProviderResource{}),
			},
			want:    nil,
			want1:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := getInventoryResourceAndID(tt.args.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("getInventoryResourceAndID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("getInventoryResourceAndID() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getInventoryResourceAndID() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
