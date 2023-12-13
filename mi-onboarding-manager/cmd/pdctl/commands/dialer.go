/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcDialer struct {
	Addr string

	SkipHostVerification bool

	CertPath   string
	KeyPath    string
	CACertPath string
	ServerName string
}

func (d *grpcDialer) Dial(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	// Load credentials
	var creds credentials.TransportCredentials
	switch {
	case d.SkipHostVerification:
		creds = insecure.NewCredentials()
	case d.CertPath != "":
		cert, err := tls.LoadX509KeyPair(d.CertPath, d.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("error loading client certificate credentials: %w", err)
		}

		cacertBytes, err := ioutil.ReadFile(d.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read CA certificate bundle from %q: %w", d.CACertPath, err)
		}
		cas := x509.NewCertPool()
		if ok := cas.AppendCertsFromPEM(cacertBytes); !ok {
			return nil, fmt.Errorf("unable to load CA certificates from %q: must be PEM formatted", d.CACertPath)
		}

		creds = credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      cas,
			ServerName:   d.ServerName,
			MinVersion:   tls.VersionTLS12,
		})
	default:
		return nil, errors.New(`required flag "insecure" XOR "cert" not set`)
	}

	// Dial without blocking
	cc, err := grpc.DialContext(ctx, d.Addr, append(opts, grpc.WithTransportCredentials(creds))...)
	if err != nil {
		return nil, fmt.Errorf("could not dial service [grpc://%s]: %w", d.Addr, err)
	}
	return cc, nil
}
