// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	computev1 "github.com/intel/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/intel/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	inv_testing "github.com/intel/infra-core/inventory/v2/pkg/testing"
	"github.com/intel/infra-core/inventory/v2/pkg/util"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/handlers/southbound"
	"github.com/intel/infra-onboarding/onboarding-manager/internal/invclient"
	om_testing "github.com/intel/infra-onboarding/onboarding-manager/internal/testing"
	pb "github.com/intel/infra-onboarding/onboarding-manager/pkg/api"
)

var (
	zlog = logging.GetLogger("Onboarding-Manager-Southbound-Testing")

	SBHandler        *southbound.SBHandler
	OMTestClient     pb.InteractiveOnboardingServiceClient
	OMTestClientConn *grpc.ClientConn
	BufconnLis       *bufconn.Listener
	InvClient        *invclient.OnboardingInventoryClient
	rbacRules        = "../../../rego/authz.rego"
)

// Internal parameters for bufconn testing.
const bufferSize = util.Megabyte

func CreateSouthboundOMClient(target string,
	bufconnLis *bufconn.Listener,
) (pb.InteractiveOnboardingServiceClient, *grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		// grpc.WithBlock(),
	}

	if bufconnLis != nil {
		opts = append(opts,
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return bufconnLis.Dial() }))
	}

	dialOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	opts = append(opts, dialOpt)

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, nil, err
	}
	southboundClient := pb.NewInteractiveOnboardingServiceClient(conn)

	return southboundClient, conn, nil
}

// Create the bufconn listener used for the client/server onboarding manager communication.
func createBufConn() {
	// https://pkg.go.dev/google.golang.org/grpc/test/bufconn#Listener
	buffer := bufferSize
	BufconnLis = bufconn.Listen(buffer)
}

// Helper function to create a southbound gRPC server for host manager.
func createOnboardingManagerSouthboundAPI() {
	sbHandler := southbound.NewSBHandlerWithListener(
		BufconnLis,
		om_testing.InvClient,
		southbound.SBHandlerConfig{
			EnableTracing: false, // be explicit
			RBAC:          rbacRules,
		},
	)
	err := sbHandler.Start()
	if err != nil {
		zlog.Fatal().Err(err).Msg("Cannot create Inventory OnboardingRM client")
	}
	SBHandler = sbHandler
}

// Helper function to start all requirements to test southbound onboarding manager client API.
func StartOnboardingManagerTestingEnvironment() {
	// Boostrap c/s connectivity using bufconn
	createBufConn()
	// Bootstrap Inventory client
	om_testing.CreateInventoryOnboardingClientForTesting()
	// Bootstrap SB server
	createOnboardingManagerSouthboundAPI()
	// Bootstrap the clients
	cli, conn, err := CreateSouthboundOMClient("", BufconnLis)
	if err != nil {
		zlog.Fatal().Err(err).Msg("Cannot create Host Manager client")
	}
	OMTestClient = cli
	OMTestClientConn = conn
}

func StopOnboardingManagerTestingEnvironment() {
	if OMTestClientConn != nil {
		OMTestClientConn.Close()
	}
	if SBHandler != nil {
		SBHandler.Stop()
	}
	om_testing.DeleteInventoryOnboardingClientForTesting()
}

// Starts all Inventory and Onboarding Manager requirements to test OM southbound client.
func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))

	policyPath := projectRoot + "/out"
	migrationsDir := projectRoot + "/out"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	StartOnboardingManagerTestingEnvironment()

	run := m.Run() // run all tests

	StopOnboardingManagerTestingEnvironment()
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func GetHostbyUUID(tb testing.TB, hostUUID string) *computev1.HostResource {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	in := &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Host{},
		},
		Filter: fmt.Sprintf("%s=%q", computev1.HostResourceFieldUuid, hostUUID),
	}

	listres, err := inv_testing.TestClients[inv_testing.RMClient].List(ctx, in)
	require.NoError(tb, err, "Get Host failed")

	resources := make([]*computev1.HostResource, 0, len(listres.Resources))
	for _, r := range listres.Resources {
		resources = append(resources, r.GetResource().GetHost())
	}

	host := resources[0]
	return host
}

//nolint:dupl //this is with TestNewSBHandler.
func TestNewSBHandler(t *testing.T) {
	type args struct {
		invClient *invclient.OnboardingInventoryClient
		config    southbound.SBHandlerConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *southbound.SBHandler
		wantErr bool
	}{
		{
			name: "NewSB handler-Success",
			args: args{
				invClient: &invclient.OnboardingInventoryClient{},
			},
			want:    &southbound.SBHandler{},
			wantErr: false,
		},
		{
			name: "NewSB handler-failure",
			args: args{
				config: southbound.SBHandlerConfig{
					ServerAddress: "abc",
				},
				invClient: &invclient.OnboardingInventoryClient{},
			},
			want:    &southbound.SBHandler{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := southbound.NewSBHandler(tt.args.invClient, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSBHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSBHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func FuzzNewSBHandler(f *testing.F) {
	f.Add("127.0.0.1:9090", true, true, "admin")
	f.Add("localhost:8080", false, false, "user")
	f.Fuzz(func(t *testing.T, inventoryAddress string, enableTracing, enableAuth bool, rbac string) {
		invClient := &invclient.OnboardingInventoryClient{}
		config := southbound.SBHandlerConfig{
			ServerAddress:    "",
			InventoryAddress: inventoryAddress,
			EnableTracing:    enableTracing,
			EnableAuth:       enableAuth,
			RBAC:             rbac,
		}

		handler, err := southbound.NewSBHandler(invClient, config)
		if err != nil {
			t.Logf("Expected error: %v", err)
		} else {
			if handler == nil {
				t.Errorf("Expected non-nil handler, got nil")
			} else {
				t.Logf("Handler created successfully: %+v", handler)
			}
		}
	})
}

//nolint:dupl //this is with TestNewSBNioHandler.
func TestNewSBNioHandler(t *testing.T) {
	type args struct {
		invClient *invclient.OnboardingInventoryClient
		config    southbound.SBHandlerNioConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *southbound.SBNioHandler
		wantErr bool
	}{
		{
			name: "NewSB handler-Success",
			args: args{
				invClient: &invclient.OnboardingInventoryClient{},
			},
			want:    &southbound.SBNioHandler{},
			wantErr: false,
		},
		{
			name: "NewSB handler-failure",
			args: args{
				config: southbound.SBHandlerNioConfig{
					ServerAddressNio: "abc",
				},
				invClient: &invclient.OnboardingInventoryClient{},
			},
			want:    &southbound.SBNioHandler{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := southbound.NewSBNioHandler(tt.args.invClient, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSBNioHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSBNioHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSBNioHandler_Start(t *testing.T) {
	sbNioHandler, err := southbound.NewSBNioHandler(om_testing.InvClient, southbound.SBHandlerNioConfig{})
	if err != nil {
		fmt.Println(err)
	}
	startErr := sbNioHandler.Start()
	if startErr != nil {
		t.Errorf("sbNioHandler.Start() = %v", startErr)
	}
}
