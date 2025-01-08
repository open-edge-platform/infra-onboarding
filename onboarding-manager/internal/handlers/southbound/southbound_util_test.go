// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel
package southbound

import (
	"net"
	"testing"

	"google.golang.org/grpc"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
)

const rbacRules = "../../../rego/authz.rego"

func TestSBHandler_Stop(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:16541")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	go func() {
		defer lis.Close()
		if err := grpcServer.Serve(lis); err != nil {
			t.Fatalf("Failed to serve: %v", err)
		}
	}()
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
		t.Run(tt.name, func(t *testing.T) {
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
