// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
//
//nolint:testpackage // Keeping the test in the same package due to dependencies on unexported fields.
package southbound

import (
	"net"
	"testing"

	"google.golang.org/grpc"

	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/invclient"
)

const rbacRules = "../../../rego/authz.rego"

func TestSBHandler_Stop(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:16541")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	//nosemgrep: go.grpc.security.grpc-server-insecure-connection.grpc-server-insecure-connection // test scenario
	grpcServer := grpc.NewServer()
	//nolint:staticcheck // Ignoring SA2002 and SA1019 as these are valid in this test scenario.
	go func() {
		defer lis.Close()
		if err := grpcServer.Serve(lis); err != nil {
			// nolint:staticcheck,govet // Ignoring SA2002 and SA1019 as these are valid in this test scenario.
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
	//nolint:staticcheck // Ignoring SA1019 as these are valid in this test scenario
	//nosemgrep: go.grpc.security.grpc-client-insecure-connection.grpc-client-insecure-connection // test scenario
	conn, conErr := grpc.Dial("localhost:13051", grpc.WithInsecure())
	if conErr != nil {
		t.Fatalf("Failed to dial server: %v", conErr)
	}
	defer conn.Close()
	type fields struct {
		invClient *invclient.OnboardingInventoryClient
		cfg       SBHandlerConfig
		lis       net.Listener
		server    *grpc.Server
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "graceful Shutdown",
			fields: fields{
				invClient: &invclient.OnboardingInventoryClient{},
				server:    grpcServer,
				cfg: SBHandlerConfig{
					RBAC: rbacRules,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			sbh := &SBHandler{
				invClient: tt.fields.invClient,
				cfg:       tt.fields.cfg,
				lis:       tt.fields.lis,
				server:    tt.fields.server,
			}
			sbh.Stop()
		})
	}
}
