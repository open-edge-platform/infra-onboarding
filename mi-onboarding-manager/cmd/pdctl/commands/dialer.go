/*
SPDX-FileCopyrightText: (C) 2023 Intel Corporation
SPDX-License-Identifier: LicenseRef-Intel
*/
package commands

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"

	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client"
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

		cacertBytes, err := os.ReadFile(d.CACertPath)
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

func NewInventoryClient(
	ctx context.Context,
	wg *sync.WaitGroup,
	addr string,
) (inv_client.InventoryClient, chan *inv_client.WatchEvents, error) {
	fmt.Println("Init Inv client")
	resourceKinds := []inv_v1.ResourceKind{
		inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
		inv_v1.ResourceKind_RESOURCE_KIND_HOST,
		inv_v1.ResourceKind_RESOURCE_KIND_OS,
	}
	eventCh := make(chan *inv_client.WatchEvents)

	cfg := inv_client.InventoryClientConfig{
		Name:                      "onboarding_manager",
		Address:                   addr,
		Events:                    eventCh,
		EnableRegisterRetry:       false,
		AbortOnUnknownClientError: true,
		ClientKind:                inv_v1.ClientKind_CLIENT_KIND_API,
		ResourceKinds:             resourceKinds,
		EnableTracing:             true,
		Wg:                        wg,
		SecurityCfg: &inv_client.SecurityConfig{
			Insecure: true,
		},
	}
	for {
		client, err := inv_client.NewInventoryClient(ctx, cfg)
		if err != nil {
			fmt.Printf("Failed to create new inventory client %v,Retry after 5 seconds", err)
			time.Sleep(timeDuration)
			return nil, nil, err
		}
		if err == nil {
			return client, eventCh, nil
		}
	}
}
