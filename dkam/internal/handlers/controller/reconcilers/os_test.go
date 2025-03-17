// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
//
//nolint:testpackage // Keeping the test in the same package due to dependencies on unexported fields.
package reconcilers

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-onboarding/dkam/internal/invclient"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	dkam_testing "github.com/open-edge-platform/infra-onboarding/dkam/testing"
	rec_v2 "github.com/open-edge-platform/orch-library/go/pkg/controller/v2"
)

const (
	tenant1 = "11111111-1111-1111-1111-111111111111"
)

var projectRoot string

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot = filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(wd))))
	policyPath := projectRoot + "/out"
	migrationsDir := projectRoot + "/out"

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

func TestOsReconcilerReconcile(t *testing.T) {
	type fields struct {
		invClient     *invclient.DKAMInventoryClient
		enableTracing bool
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ReconcilerID]
	}
	dkam_testing.CreateInventoryDKAMClientForTesting()
	t.Cleanup(func() {
		dkam_testing.DeleteInventoryDKAMClientForTesting()
	})
	pvc := config.PVC
	config.PVC = "data"
	defer func() {
		config.PVC = pvc
	}()
	osre := inv_testing.CreateOsWithArgs(t, "", "profile:profile",
		osv1.SecurityFeature_SECURITY_FEATURE_NONE, osv1.OsType_OS_TYPE_MUTABLE)
	testRequest := rec_v2.Request[ReconcilerID]{
		ID: WrapReconcilerID(osre.TenantId, osre.ResourceId),
	}
	testRequest1 := rec_v2.Request[ReconcilerID]{
		ID: WrapReconcilerID(osre.TenantId, "os-12345"),
	}
	rawFileName := strings.TrimSuffix(osre.ProfileName, ".img") + ".raw.gz"
	expectedFilePath := config.PVC + "/OSImage/" + osre.Sha256 + "/" + rawFileName
	err := os.MkdirAll(filepath.Dir(expectedFilePath), 0o755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}
	file, err := os.Create(expectedFilePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	file.Close()
	//
	filePath := config.PVC + "/OSArtifacts/" + osre.ResourceId + "/installer.sh"
	dir := filepath.Dir(filePath)
	mkerr := os.MkdirAll(dir, 0o755)
	if mkerr != nil {
		t.Fatalf("Failed to create directories: %v", mkerr)
	}
	file, cerr := os.Create(filePath)
	if cerr != nil {
		t.Fatalf("Failed to create file: %v", cerr)
	}
	defer file.Close()
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("File was not created")
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "TestOsReconciler with Mutable os",
			fields: fields{
				invClient:     dkam_testing.InvClient,
				enableTracing: true,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
		},
		{
			name: "Test Case",
			fields: fields{
				invClient:     dkam_testing.InvClient,
				enableTracing: false,
			},
			args: args{
				ctx:     context.Background(),
				request: testRequest1,
			},
		},
	}
	defer func() {
		err := os.Remove(expectedFilePath)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("Failed to remove file: %v", err)
		}
		err = os.RemoveAll(filepath.Dir(expectedFilePath))
		if err != nil {
			t.Fatalf("Failed to clean up directories: %v", err)
		}
		os.Remove(expectedFilePath)
		os.Remove(dir + "/installer.sh")
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			osr := &OsReconciler{
				invClient:     tt.fields.invClient,
				enableTracing: tt.fields.enableTracing,
			}
			osr.Reconcile(tt.args.ctx, tt.args.request)
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
		request rec_v2.Request[ReconcilerID]
	}
	dkam_testing.CreateInventoryDKAMClientForTesting()
	t.Cleanup(func() {
		dkam_testing.DeleteInventoryDKAMClientForTesting()
	})
	osre := inv_testing.CreateOsWithArgs(t, "", "profile:profile", osv1.SecurityFeature_SECURITY_FEATURE_NONE,
		osv1.OsType_OS_TYPE_IMMUTABLE)
	testRequest := rec_v2.Request[ReconcilerID]{
		ID: WrapReconcilerID(osre.TenantId, osre.ResourceId),
	}
	expectedFilePath := config.DownloadPath + "/" + "tiberos.raw.xz"
	err := os.MkdirAll(filepath.Dir(expectedFilePath), 0o755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}
	file, err := os.Create(expectedFilePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	file.Close()

	err = dkam_testing.CopyFile(
		filepath.Join(projectRoot, "test", "testdata", "example-manifest-internal-rs.yaml"),
		filepath.Join(config.DownloadPath, "tmp", config.ReleaseVersion+".yaml"))
	require.NoError(t, err)
	defer func() {
		os.RemoveAll(filepath.Join(config.DownloadPath, "tmp"))
	}()

	tests := []struct {
		name   string
		fields fields
		args   args
		want   rec_v2.Directive[ReconcilerID]
	}{
		{
			name: "test case with immutable os",
			fields: fields{
				invClient:     dkam_testing.InvClient,
				enableTracing: true,
			},
			args: args{
				ctx:     context.Background(),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
	}
	defer func() {
		os.RemoveAll(config.PVC)
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osr := &OsReconciler{
				invClient:     tt.fields.invClient,
				enableTracing: tt.fields.enableTracing,
			}
			if got := osr.Reconcile(tt.args.ctx, tt.args.request); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OsReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOsReconcilerReconcile_DownloadOs_Err(t *testing.T) {
	type fields struct {
		invClient     *invclient.DKAMInventoryClient
		enableTracing bool
	}
	type args struct {
		ctx     context.Context
		request rec_v2.Request[ReconcilerID]
	}
	dkam_testing.CreateInventoryDKAMClientForTesting()
	t.Cleanup(func() {
		dkam_testing.DeleteInventoryDKAMClientForTesting()
	})
	osre := inv_testing.CreateOsWithArgs(t, "", "profile:profile", osv1.SecurityFeature_SECURITY_FEATURE_NONE,
		osv1.OsType_OS_TYPE_MUTABLE)
	testRequest := rec_v2.Request[ReconcilerID]{
		ID: WrapReconcilerID(osre.TenantId, osre.ResourceId),
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "TestOsReconciler with Mutable os",
			fields: fields{
				invClient:     dkam_testing.InvClient,
				enableTracing: true,
			},
			args: args{
				ctx:     context.TODO(),
				request: testRequest,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			osr := &OsReconciler{
				invClient:     tt.fields.invClient,
				enableTracing: tt.fields.enableTracing,
			}
			osr.Reconcile(tt.args.ctx, tt.args.request)
		})
	}
}

// Fuzz test.
func FuzzReconcileOs(f *testing.F) {
	f.Add("ec426b10")

	f.Fuzz(func(t *testing.T, id string) {
		dkam_testing.CreateInventoryDKAMClientForTesting()
		t.Cleanup(func() {
			dkam_testing.DeleteInventoryDKAMClientForTesting()
		})

		if id == "" || len(id) < 5 {
			t.Skip("Skip as osname or Id is empty")
			return
		}

		id = "os-" + id

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// mutex.Lock()
		osre := inv_testing.CreateOsWithArgs(t, "", "profile:profile",
			osv1.SecurityFeature_SECURITY_FEATURE_NONE, osv1.OsType_OS_TYPE_MUTABLE)

		request := rec_v2.Request[ReconcilerID]{
			ID: WrapReconcilerID(tenant1, id),
		}
		osr := &OsReconciler{
			invClient:     dkam_testing.InvClient,
			enableTracing: false,
		}

		osr.reconcileOs(ctx, request, osre)
	})
}
