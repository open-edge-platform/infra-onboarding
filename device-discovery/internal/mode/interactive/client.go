// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interactive

import (
	"context"
	"crypto/x509"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	pb "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/api/onboardingmgr/v1"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	"device-discovery/internal/config"
)

// Client handles interactive (manual) device onboarding with JWT authentication.
type Client struct {
	address        string
	port           int
	mac            string
	ipAddress      string
	uuid           string
	serial         string
	caCertPath     string
	accessTokenPath string
}

// NewClient creates a new interactive mode client.
func NewClient(address string, port int, mac, ipAddress, uuid, serial, caCertPath, accessTokenPath string) *Client {
	return &Client{
		address:        address,
		port:           port,
		mac:            mac,
		ipAddress:      ipAddress,
		uuid:           uuid,
		serial:         serial,
		caCertPath:     caCertPath,
		accessTokenPath: accessTokenPath,
	}
}

// Onboard performs interactive device onboarding using JWT authentication.
// This requires that the user has already authenticated and obtained an access token.
func (c *Client) Onboard(ctx context.Context) error {
	// Load the CA certificate
	caCert, err := os.ReadFile(c.caCertPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}
	fmt.Println("caCert: ", caCert)
	
	// Create a certificate pool from the CA certificate
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to append CA certificate to cert pool")
	}

	// Create the credentials using the certificate pool
	creds := credentials.NewClientTLSFromCert(certPool, "")

	// Read JWT token from file
	jwtToken, err := os.ReadFile(c.accessTokenPath)
	if err != nil {
		return fmt.Errorf("failed to read JWT token from file: %v", err)
	}
	// Convert JWT token to string and trim whitespace
	tokenString := strings.TrimSpace(string(jwtToken))

	target := fmt.Sprintf("%s:%d", c.address, c.port)
	conn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(
			oauth.TokenSource{
				TokenSource: oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: tokenString}, // Send the access token as part of the HTTP Authorization header
				),
			},
		),
	)
	if err != nil {
		return fmt.Errorf("could not dial server %s: %v", target, err)
	}
	defer conn.Close()
	fmt.Println("Dial Complete")

	cli := pb.NewInteractiveOnboardingServiceClient(conn)
	
	// Create a NodeData object
	nodeData := &pb.NodeData{
		Hwdata: []*pb.HwData{
			{
				MacId:     c.mac,
				SutIp:     c.ipAddress,
				Uuid:      c.uuid,
				Serialnum: c.serial,
			},
		},
	}
	
	// Create a NodeRequest object and set the Payload field
	nodeRequest := &pb.CreateNodesRequest{
		Payload: []*pb.NodeData{nodeData},
	}
	
	// Call the gRPC endpoint with the NodeRequest
	nodeResponse, err := cli.CreateNodes(ctx, nodeRequest)
	if err != nil {
		return fmt.Errorf("could not call gRPC endpoint for server %s: %v", target, err)
	}

	// Check if the ProjectId field is empty
	if nodeResponse.ProjectId == "" {
		return fmt.Errorf("received empty Project ID")
	}

	// Save the Project ID to a file
	if err := config.SaveToFile(config.ProjectIDPath, nodeResponse.ProjectId); err != nil {
		return fmt.Errorf("failed to save Project ID to file: %v", err)
	}
	
	return nil
}

// OnboardWithRetry performs interactive onboarding with retry logic.
// It will attempt up to maxRetries times with exponential backoff.
func (c *Client) OnboardWithRetry(ctx context.Context) error {
	maxRetries := 3
	retryDelay := 2 * time.Second // Fixed delay between retries

	for retries := 0; retries < maxRetries; retries++ {
		err := c.Onboard(ctx)
		if err == nil {
			return nil
		}

		// Log the error and retry info
		fmt.Printf("There was an error in updating the edge-node details with the onboarding manager: %v\n", err)
		if retries < maxRetries-1 {
			fmt.Printf("Retrying update... attempt %d of %d\n", retries+2, maxRetries) // retries+2 to show next attempt
			time.Sleep(retryDelay + time.Duration(rand.Intn(1000))*time.Millisecond)   // Slight random jitter
		}
	}
	
	return fmt.Errorf("max retries reached")
}
