// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package connection

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// CreateSecureConnection creates a secure gRPC connection with TLS.
// This is used by both interactive and non-interactive modes.
func CreateSecureConnection(ctx context.Context, target string, caCertPath string) (*grpc.ClientConn, error) {
	// Load the CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %v", err)
	}

	// Create a certificate pool from the CA certificate
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate to cert pool")
	}

	// Create the credentials using the certificate pool
	creds := credentials.NewClientTLSFromCert(certPool, "")

	// Create the gRPC connection with TLS credentials
	conn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
