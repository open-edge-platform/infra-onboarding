// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package reconcilers_test

import (
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-onboarding/dkam/internal/handlers/controller/reconcilers"
	rec_v2 "github.com/open-edge-platform/orch-library/go/pkg/controller/v2"
)

func TestHandleInventoryError(t *testing.T) {
	type args struct {
		err     error
		request rec_v2.Request[reconcilers.ReconcilerID]
	}
	testRequest := rec_v2.Request[reconcilers.ReconcilerID]{
		ID: reconcilers.WrapReconcilerID(client.FakeTenantID, "test-id"),
	}
	tests := []struct {
		name string
		args args
		want rec_v2.Directive[reconcilers.ReconcilerID]
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
		t.Run(tt.name, func(_ *testing.T) {
			reconcilers.HandleInventoryError(tt.args.err, tt.args.request)
		})
	}
}

func TestHandleProvisioningError(t *testing.T) {
	type args struct {
		err     error
		request rec_v2.Request[reconcilers.ReconcilerID]
	}
	testRequest := rec_v2.Request[reconcilers.ReconcilerID]{
		ID: reconcilers.WrapReconcilerID(client.FakeTenantID, "test-id"),
	}
	retryMinDelay := 1 * time.Second
	retryMaxDelay := 60 * time.Second
	tests := []struct {
		name string
		args args
		want rec_v2.Directive[reconcilers.ReconcilerID]
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
			want: testRequest.Retry(status.Error(codes.Internal, "internal error")).With(rec_v2.ExponentialBackoff(retryMinDelay,
				retryMaxDelay)),
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
				err: inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS,
					"Prod workflow started, waiting for it to complete"),
				request: testRequest,
			},
			want: testRequest.Ack(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			reconcilers.HandleProvisioningError(tt.args.err, tt.args.request)
		})
	}
}
