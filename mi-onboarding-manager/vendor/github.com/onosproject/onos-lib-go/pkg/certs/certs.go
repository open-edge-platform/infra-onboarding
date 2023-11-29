// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
)

// HandleCertPaths is a common function for clients and servers like admin/net-changes for
// handling certificate args if given, or else loading defaults
func HandleCertPaths(caPath string, keyPath string, certPath string, insecure bool) ([]grpc.DialOption, error) {
	var opts = []grpc.DialOption{}
	var cert tls.Certificate
	var err error
	if keyPath != Client1Key && keyPath != "" &&
		certPath != Client1Crt && certPath != "" {
		cert, err = tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}

	} else {
		// Load default Certificates
		cert, err = tls.X509KeyPair([]byte(DefaultClientCrt), []byte(DefaultClientKey))
		if err != nil {
			return nil, err
		}
	}
	var clientCAs *x509.CertPool

	if caPath == "" {
		clientCAs, err = GetCertPoolDefault()
	} else {
		clientCAs, err = GetCertPool(caPath)
	}
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientCAs:          clientCAs,
		InsecureSkipVerify: insecure,
	}
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

	return opts, nil
}

// GetCertPoolDefault load the default ONF Cert Authority
func GetCertPoolDefault() (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(OnfCaCrt)); !ok {
		return nil, fmt.Errorf("failed to append default ONF CA certificate")
	}
	return certPool, nil
}

// GetCertPool loads the Certificate Authority from the given path
func GetCertPool(CaPath string) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(CaPath)
	if err != nil {
		return nil, err
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("failed to append CA certificate from %s", CaPath)
	}
	return certPool, nil
}
