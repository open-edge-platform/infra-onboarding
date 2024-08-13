// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package southbound

import (
	"net"
	"reflect"
	"sync"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/invclient"
	"google.golang.org/grpc"
)

func TestNewSBHandler(t *testing.T) {
	type args struct {
		invClient *invclient.DKAMInventoryClient
		config    SBHandlerConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *SBHandler
		wantErr bool
	}{
		{
			name: "NewSB handler-Success",
			args: args{
				invClient: &invclient.DKAMInventoryClient{},
			},
			want:    &SBHandler{},
			wantErr: false,
		},
		{
			name: "NewSB handler-failure",
			args: args{
				config: SBHandlerConfig{
					ServerAddress: "abc",
				},
				invClient: &invclient.DKAMInventoryClient{},
			},
			want:    &SBHandler{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSBHandler(tt.args.invClient, tt.args.config)
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
		invClient *invclient.DKAMInventoryClient
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
				invClient: &invclient.DKAMInventoryClient{},
				server:    grpcServer,
				cfg: SBHandlerConfig{
					RBAC: "",
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

func TestSBHandler_Start(t *testing.T) {
	type fields struct {
		invClient *invclient.DKAMInventoryClient
		cfg       SBHandlerConfig
		wg        *sync.WaitGroup
		lis       net.Listener
		server    *grpc.Server
	}
	grpcServer := grpc.NewServer()
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "Start Test Case",
			fields:  fields{
				invClient:&invclient.DKAMInventoryClient{},
				cfg: SBHandlerConfig{
					RBAC: "",
				},
				server:    grpcServer,
			},
			wantErr: false,
		},
		{
			name:    "Start Test Case Error",
			fields:  fields{
				invClient:nil,
				cfg: SBHandlerConfig{
					RBAC: "",
				},
				server:    grpcServer,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sbh := &SBHandler{
				invClient: tt.fields.invClient,
				cfg:       tt.fields.cfg,
				wg:        tt.fields.wg,
				lis:       tt.fields.lis,
				server:    tt.fields.server,
			}
			if err := sbh.Start(); (err != nil) != tt.wantErr {
				t.Errorf("SBHandler.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
