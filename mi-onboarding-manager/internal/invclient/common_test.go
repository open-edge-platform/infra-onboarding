// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package invclient

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	network_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/network/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
)

func Test_getInventoryResourceAndID(t *testing.T) {
	type args struct {
		resource proto.Message
	}
	hostResource := &computev1.HostResource{}
	hostResCopy := proto.Clone(hostResource)
	hostStorageResource := &computev1.HoststorageResource{}
	hostStorageResCopy := proto.Clone(hostStorageResource)
	hostSubResource := &computev1.HostusbResource{}
	hostSubResCopy := proto.Clone(hostSubResource)
	hostNicResource := &computev1.HostnicResource{}
	hostNicResCopy := proto.Clone(hostNicResource)
	hostgpuResource := &computev1.HostgpuResource{}
	hostgpuResourceCopy := proto.Clone(hostgpuResource)
	networkResource := &network_v1.IPAddressResource{}
	networkResourceCopy := proto.Clone(networkResource)
	operatingSystemResource := &osv1.OperatingSystemResource{}
	operatingSystemResourceCopy := proto.Clone(operatingSystemResource)
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
				resource: hostResCopy,
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Host Storage Resource Test",
			args: args{
				resource: hostStorageResCopy,
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Host USB Resource Test",
			args: args{
				resource: hostSubResCopy,
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Host GPU Resource Test",
			args: args{
				resource: hostgpuResourceCopy,
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Network Resource Test",
			args: args{
				resource: networkResourceCopy,
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "Operating System Resource Test",
			args: args{
				resource: operatingSystemResourceCopy,
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
		},
		{
			name: "hostNic Resource Test",
			args: args{
				resource: hostNicResCopy,
			},
			want:    &inv_v1.Resource{},
			want1:   "",
			wantErr: false,
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
