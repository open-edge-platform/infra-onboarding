// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func NewClientConn(authority string, tls bool) (*grpc.ClientConn, error) {
	var creds grpc.DialOption
	if tls {
		creds = grpc.WithTransportCredentials(credentials.NewTLS(nil))
	} else {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	//nolint:staticcheck // SA1019 grpc.Dial is deprecated.
	conn, err := grpc.Dial(authority, creds, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		return nil, errors.Wrap(err, "dial tinkerbell server")
	}

	return conn, nil
}
