// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"errors"
	"testing"

	rec_v2 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.lib-go/pkg/controller/v2"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandleInventoryError(t *testing.T) {
	type args struct {
		err     error
		request rec_v2.Request[ResourceID]
	}
	testRequest := rec_v2.Request[ResourceID]{
		ID: ResourceID("test-id"),
	}
	tests := []struct {
		name string
		args args
		want rec_v2.Directive[ResourceID]
	}{
		{
			name: "HandleInventoryError Test Case - Non-gRPC Error",
			args: args{
				err:     errors.New("standard error"),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "HandleInventoryError Test Case - gRPC Error",
			args: args{
				err:     status.Error(codes.NotFound, "not found"),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "HandleInventoryError Test Case - No Error",
			args: args{
				err:     nil,
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleInventoryError(tt.args.err, tt.args.request)
		})
	}
}

func TestHandleProvisioningError(t *testing.T) {
	type args struct {
		err     error
		request rec_v2.Request[ResourceID]
	}
	testRequest := rec_v2.Request[ResourceID]{
		ID: ResourceID("test-id"),
	}
	tests := []struct {
		name string
		args args
		want rec_v2.Directive[ResourceID]
	}{
		{
			name: "Non-gRPC Error",
			args: args{
				err:     errors.New("standard error"),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
		{
			name: "gRPC Error - Aborted",
			args: args{
				err:     status.Error(codes.Aborted, "aborted"),
				request: testRequest,
			},
			want: testRequest.Fail(status.Error(codes.Aborted, "aborted")),
		},
		{
			name: "Other gRPC Error",
			args: args{
				err:     status.Error(codes.Internal, "internal error"),
				request: testRequest,
			},
			want: testRequest.Retry(status.Error(codes.Internal, "internal error")).With(rec_v2.ExponentialBackoff(retryMinDelay, retryMaxDelay)),
		},
		{
			name: "No Error",
			args: args{
				err:     nil,
				request: testRequest,
			},
			want: nil,
		},
		{
			name: "HandleInventoryError Test Case - In Progress Error",
			args: args{
				err:     inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "Prod workflow started, waiting for it to complete"),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleProvisioningError(tt.args.err, tt.args.request)
		})
	}
}
