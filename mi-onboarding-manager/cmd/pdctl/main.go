/*
SPDX-FileCopyrightText: (C) 2023 Intel Corporation
SPDX-License-Identifier: LicenseRef-Intel
*/
package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/cmd/pdctl/commands"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

var (
	clientName = "pdctl"
	zlog       = logging.GetLogger(clientName)
)

// type grpcDialer struct {
// 	Addr string

// 	SkipHostVerification bool

// 	CertPath   string
// 	KeyPath    string
// 	CACertPath string
// 	ServerName string
// }

// func (d *grpcDialer) Dial(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
// 	// Load credentials
// 	var creds credentials.TransportCredentials
// 	switch {
// 	case d.SkipHostVerification:
// 		creds = insecure.NewCredentials()
// 	case d.CertPath != "":
// 		cert, err := tls.LoadX509KeyPair(d.CertPath, d.KeyPath)
// 		if err != nil {
// 			return nil, fmt.Errorf("error loading client certificate credentials: %w", err)
// 		}

// 		cacertBytes, err := ioutil.ReadFile(d.CACertPath)
// 		if err != nil {
// 			return nil, fmt.Errorf("unable to read CA certificate bundle from %q: %w", d.CACertPath, err)
// 		}
// 		cas := x509.NewCertPool()
// 		if ok := cas.AppendCertsFromPEM(cacertBytes); !ok {
// 			return nil, fmt.Errorf("unable to load CA certificates from %q: must be PEM formatted", d.CACertPath)
// 		}

// 		creds = credentials.NewTLS(&tls.Config{
// 			Certificates: []tls.Certificate{cert},
// 			RootCAs:      cas,
// 			ServerName:   d.ServerName,
// 			MinVersion:   tls.VersionTLS12,
// 		})
// 	default:
// 		return nil, errors.New(`required flag "insecure" XOR "cert" not set`)
// 	}

// 	// Dial without blocking
// 	cc, err := grpc.DialContext(ctx, d.Addr, append(opts, grpc.WithTransportCredentials(creds))...)
// 	if err != nil {
// 		return nil, fmt.Errorf("could not dial service [grpc://%s]: %w", d.Addr, err)
// 	}
// 	return cc, nil
// }

// Version is set with linker flags at build time.
var Version string

func main() {
	// Root command and persistent flags
	rootCmd := &cobra.Command{
		Use:     "pdctl",
		Short:   "pdctl - a CLI for Intel Platform Director",
		Version: Version,
		Long: `pdctl is a CLI to access and manage different instances of Platform Director
   for on-premise and cloud hosted instances`,
	}
	verbose := rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")

	// Setup global logger

	// Add subcommands
	rootCmd.AddCommand(commands.HostResourceCmd())
	rootCmd.AddCommand(commands.InstanceResourceCmd())
	rootCmd.AddCommand(commands.HostResCmds())
	rootCmd.AddCommand(commands.InstanceResCmds())
	rootCmd.AddCommand(commands.OsCmds())

	// Execute CLI
	if err := rootCmd.Execute(); err != nil {
		if *verbose {
			zlog.Debug().Msgf("%v", err)
		}
		os.Exit(1)
	}
}
