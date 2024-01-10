// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

// The OAM gRPC server implemented in this package offers a gRPC endpoint implementing the gRPC health checking
// protocol. This together with the capabilities of k8s can be used to implement liveness/readiness meachanism
// for all our pods (not only for Inventory). Note below that only the Check API has been implemented.
package oam

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

const (
	OamServerAddress            = "oamServerAddress"
	OamServerAddressDescription = "The OAM server address to serve on. It should have the following format <IP address>:<port>."
)

var zlog = logging.GetLogger("MIOAMgRPC")

// Store the readiness status.
type OAM struct {
	ready atomic.Bool
}

// Initialize a new OAM structure.
func NewOAM() *OAM {
	return &OAM{}
}

// Return the readiness status.
func (o *OAM) isReady() bool {
	return o.ready.Load()
}

// Update the ready state.
func (o *OAM) SetReady(ready bool) {
	o.ready.Store(ready)
}

// Healt check handler.
func (o *OAM) Check(_ context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	zlog.Trace().Msgf("Serving the Check request for health check")
	if o.isReady() {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVING,
		}, nil
	}
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
	}, nil
}

// Streaming API for watcher registration.
func (o *OAM) Watch(_ *grpc_health_v1.HealthCheckRequest, _ grpc_health_v1.Health_WatchServer) error {
	zlog.Trace().Msgf("Serving the Watch request for health check")
	return status.Errorf(codes.Unimplemented, "unimplemented")
}
