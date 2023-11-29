// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"

	"github.com/onosproject/onos-lib-go/pkg/certs"

	"google.golang.org/grpc/credentials"

	"google.golang.org/grpc"
)

// Service provides service-specific registration for grpc services.
type Service interface {
	Register(s *grpc.Server)
}

// Server provides NB gNMI server for onos-lib-go.
type Server struct {
	cfg      *ServerConfig
	services []Service
}

// ServerConfig comprises a set of server configuration options.
type ServerConfig struct {
	CaPath   *string
	KeyPath  *string
	CertPath *string
	Port     int16
	Insecure bool
}

// NewServer initializes server using the supplied configuration.
func NewServer(cfg *ServerConfig) *Server {
	return &Server{
		services: []Service{},
		cfg:      cfg,
	}
}

// NewServerConfig creates a server config created with the specified end-point security details.
func NewServerConfig(caPath string, keyPath string, certPath string) *ServerConfig {
	return &ServerConfig{
		Port:     5150,
		Insecure: true,
		CaPath:   &caPath,
		KeyPath:  &keyPath,
		CertPath: &certPath,
	}
}

// AddService adds a Service to the server to be registered on Serve.
func (s *Server) AddService(r Service) {
	s.services = append(s.services, r)
}

// Serve starts the NB server.
func (s *Server) Serve(started func(string)) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return err
	}

	tlsCfg := &tls.Config{}

	if *s.cfg.CertPath == "" && *s.cfg.KeyPath == "" {
		// Load default Certificates
		clientCerts, err := tls.X509KeyPair([]byte(certs.DefaultLocalhostCrt), []byte(certs.DefaultLocalhostKey))
		if err != nil {
			fmt.Println("Error loading default certs")
			return err
		}
		tlsCfg.Certificates = []tls.Certificate{clientCerts}
	} else {
		//log.Infof("Loading certs: %s %s", *s.cfg.CertPath, *s.cfg.KeyPath)
		clientCerts, err := tls.LoadX509KeyPair(*s.cfg.CertPath, *s.cfg.KeyPath)
		if err != nil {
			fmt.Println("Error loading default certs")
		}
		tlsCfg.Certificates = []tls.Certificate{clientCerts}
	}

	if s.cfg.Insecure {
		// RequestClientCert will ask client for a certificate but won't
		// require it to proceed. If certificate is provided, it will be
		// verified.
		tlsCfg.ClientAuth = tls.RequestClientCert
	} else {
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if *s.cfg.CaPath == "" {
		//log.Info("Loading default CA onfca")
		tlsCfg.ClientCAs = getCertPoolDefault()
	} else {
		tlsCfg.ClientCAs = getCertPool(*s.cfg.CaPath)
	}

	opts := []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}
	server := grpc.NewServer(opts...)
	for i := range s.services {
		s.services[i].Register(server)
	}
	started(lis.Addr().String())

	//log.Infof("Starting RPC server on address: %s", lis.Addr().String())
	return server.Serve(lis)
}

func getCertPoolDefault() *x509.CertPool {
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(certs.OnfCaCrt)); !ok {
		fmt.Println("failed to append CA certificates")
	}
	return certPool
}

func getCertPool(CaPath string) *x509.CertPool {
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(CaPath)
	if err != nil {
		fmt.Println("could not read ", CaPath, err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		fmt.Println("failed to append CA certificates")
	}
	return certPool
}
