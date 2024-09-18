// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	rec_v2 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.lib-go/pkg/controller/v2"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/invclient"
	dkam_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/testing"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/testing"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(wd))))
	policyPath := projectRoot + "/build"
	migrationsDir := projectRoot + "/build"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run()
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestNewOsReconciler(t *testing.T) {
	type args struct {
		c             *invclient.DKAMInventoryClient
		enableTracing bool
	}
	tests := []struct {
		name string
		args args
		want *OsReconciler
	}{
		{
			name: "Positive- creates a new OsReconciler instance with the given InventoryClient",
			args: args{
				c: &invclient.DKAMInventoryClient{},
			},
			want: &OsReconciler{
				invClient: &invclient.DKAMInventoryClient{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewOsReconciler(tt.args.c, tt.args.enableTracing); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOsReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOsReconciler_Reconcile(t *testing.T) {
	type fields struct {
		invClient     *invclient.DKAMInventoryClient
		enableTracing bool
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ResourceID]
	}
	dkam_testing.CreateInventoryDKAMClientForTesting()
	t.Cleanup(func() {
		dkam_testing.DeleteInventoryDKAMClientForTesting()
	})
	testRequest := rec_v2.Request[ResourceID]{
		ID: ResourceID("test-id"),
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ResourceID]
	}{
		{
			name: "TestOsReconciler_ReconcileWithErrorFetchingResource",
			fields: fields{
				invClient: dkam_testing.InvClient,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "Test Os reconciler -reconcileWith successful resource Fetch",
			fields: fields{
				invClient:     dkam_testing.InvClient,
				enableTracing: true,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osr := &OsReconciler{
				invClient:     tt.fields.invClient,
				enableTracing: tt.fields.enableTracing,
			}
			if got := osr.Reconcile(tt.args.ctx, tt.args.request); reflect.DeepEqual(got, tt.want) {
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
			name: "Test Case",
			args: args{
				originalOSRes: &osv1.OperatingSystemResource{},
				updatedOSRes:  &osv1.OperatingSystemResource{},
				fieldmask:     &fieldmaskpb.FieldMask{},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Invalid fieldmask",
			args: args{
				originalOSRes: &osv1.OperatingSystemResource{},
				updatedOSRes:  &osv1.OperatingSystemResource{},
				fieldmask:     &fieldmaskpb.FieldMask{Paths: []string{"nonexistent_field"}},
			},
			want:    false,
			wantErr: true,
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
