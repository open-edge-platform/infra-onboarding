// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

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

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/util"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/handlers/southbound"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/onboarding"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var (
	clientName     = "TestOnboardingInventoryClient"
	defaultTimeout = 120 * time.Second
	zlog           = logging.GetLogger("Onboarding-Manager-Southbound-Testing")

	SBHandler        *southbound.SBHandler
	OMTestClient     pb.NodeArtifactServiceNBClient
	OMTestClientConn *grpc.ClientConn
	BufconnLis       *bufconn.Listener
	InvClient        *invclient.OnboardingInventoryClient
)

// Internal parameters for bufconn testing.
const bufferSize = util.Megabyte

func CreateSouthboundOMClient(target string,
	bufconnLis *bufconn.Listener,
) (pb.NodeArtifactServiceNBClient, *grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
	}

	if bufconnLis != nil {
		opts = append(opts,
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return bufconnLis.Dial() }))
	}

	dialOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	opts = append(opts, dialOpt)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, nil, err
	}
	southboundClient := pb.NewNodeArtifactServiceNBClient(conn)

	return southboundClient, conn, nil
}

// CreateNetworkingClient is an helper function to create a new Networking Client.
func createInventoryOnboardingClientForTesting() {
	resourceKinds := []inv_v1.ResourceKind{
		inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
		inv_v1.ResourceKind_RESOURCE_KIND_HOST,
		inv_v1.ResourceKind_RESOURCE_KIND_OS,
	}
	err := inv_testing.CreateClient(clientName, inv_v1.ClientKind_CLIENT_KIND_RESOURCE_MANAGER, resourceKinds, "")
	if err != nil {
		zlog.Fatal().Err(err).Msg("Cannot create Inventory OnboardingRM client")
	}

	InvClient, err = invclient.NewOnboardingInventoryClient(inv_testing.TestClients[clientName],
		inv_testing.TestClientsEvents[clientName])
	if err != nil {
		zlog.Fatal().Err(err).Msg("Cannot create Inventory OnboardingRM client")
	}
}

// Create the bufconn listener used for the client/server onboarding manager communication.
func createBufConn() {
	// https://pkg.go.dev/google.golang.org/grpc/test/bufconn#Listener
	buffer := bufferSize
	BufconnLis = bufconn.Listen(buffer)
}

// Helper function to create a southbound gRPC server for host manager.
func createOnboardingManagerSouthboundAPI() {
	sbHandler := southbound.NewSBHandlerWithListener(BufconnLis, InvClient, southbound.SBHandlerConfig{
		EnableTracing: false, // be explicit
	})
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
	createInventoryOnboardingClientForTesting()
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
	if InvClient != nil {
		InvClient.Close()
	}
}

// Starts all Inventory and Onboarding Manager requirements to test OM southbound client.
func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))

	policyPath := projectRoot + "/build"
	migrationsDir := projectRoot + "/build"

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
			name: "Test Case",
			args: args{
				invClient: &invclient.OnboardingInventoryClient{
					Client: &onboarding.MockInventoryClient{},
				},
			},
			want:    &southbound.SBHandler{},
			wantErr: false,
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

