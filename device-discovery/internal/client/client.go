// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	pb "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/api/onboardingmgr/v1"
	"golang.org/x/oauth2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	"device-discovery/internal/config"
)

// createSecureConnection creates a secure gRPC connection with TLS.
func createSecureConnection(ctx context.Context, target string, caCertPath string) (*grpc.ClientConn, error) {
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

// GrpcStreamClient establishes a stream with the onboarding service.
func GrpcStreamClient(ctx context.Context, address string, port int, mac string, uuid string, serial string, ipAddress string, caCertPath string) (string, string, error, bool) {
	var fallback = false
	target := fmt.Sprintf("%s:%d", address, port)
	conn, err := createSecureConnection(ctx, target, caCertPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect: %v", err), fallback
	}
	defer conn.Close()

	cli := pb.NewNonInteractiveOnboardingServiceClient(conn)

	// Establish a stream with the server
	stream, err := cli.OnboardNodeStream(ctx)
	if err != nil {
		return "", "", fmt.Errorf("could not create stream: %v", err), fallback
	}
	defer stream.CloseSend()

	// Send a request over the stream
	request := &pb.OnboardNodeStreamRequest{
		MacId:     mac,
		Uuid:      uuid,
		Serialnum: serial,
		HostIp:    ipAddress,
	}

	// Receiving response from server
	var backoff time.Duration = 2 * time.Second
	maxBackoff := 32 * time.Second
	for {
		if err := stream.Send(request); err != nil {
			return "", "", fmt.Errorf("could not send data to server: %v", err), fallback
		}
		// Ensure stream is not nil
		if stream == nil {
			return "", "", fmt.Errorf("stream is nil"), fallback
		}

		// Receive response from the server
		resp, err := stream.Recv()
		if err == io.EOF {
			return "", "", fmt.Errorf("stream closed by server"), fallback
		}
		if err != nil {
			return "", "", fmt.Errorf("error receiving response from server: %v", err), fallback
		}

		// Ensure the response is not nil
		if resp == nil {
			return "", "", fmt.Errorf("received nil response from server"), fallback
		}

		// Handle different node states
		if resp.Status.Code == int32(codes.OK) {
			switch resp.NodeState {
			case pb.OnboardNodeStreamResponse_NODE_STATE_REGISTERED:
				fmt.Println("Edge node registered. Waiting for the edge node to become ready for onboarding...")

				// Sleep for a randomized backoff duration
				time.Sleep(backoff + time.Duration(rand.Intn(1000))*time.Millisecond)

				// Double the backoff time, but cap it at maxBackoff
				backoff *= 2
				if backoff > maxBackoff {
					backoff = 2 * time.Second
				}

			case pb.OnboardNodeStreamResponse_NODE_STATE_ONBOARDED:
				clientID := resp.ClientId
				clientSecret := resp.ClientSecret
				projectID := resp.ProjectId

				// Ensure the Project ID is not empty
				if projectID == "" {
					return "", "", fmt.Errorf("received empty Project ID"), fallback
				}

				// Save the Project ID to a file
				if err := config.SaveToFile(config.ProjectIDPath, projectID); err != nil {
					return "", "", fmt.Errorf("failed to save Project ID to file: %v", err), fallback
				}

				// Ensure both clientID and clientSecret are not empty
				if clientID == "" || clientSecret == "" {
					return "", "", fmt.Errorf("received empty clientID or clientSecret"), fallback
				}
				return clientID, clientSecret, nil, fallback

			case pb.OnboardNodeStreamResponse_NODE_STATE_UNSPECIFIED:
				return "", "", fmt.Errorf("edge node state is unspecified or unknown"), fallback

			default:
				return "", "", fmt.Errorf("unknown node state: %v", resp.NodeState), fallback
			}
		} else if resp.Status.Code == int32(codes.NotFound) {
			fallback = true
			return "", "", fmt.Errorf(resp.Status.Message), fallback
		} else {
			return "", "", fmt.Errorf(resp.Status.Message), fallback
		}
	}
}

// GrpcInfraOnboardNodeJWT performs interactive onboarding using JWT authentication.
func GrpcInfraOnboardNodeJWT(ctx context.Context, address string, port int, mac string, ip string, uuid string, serial string, caCertPath string, accessTokenPath string) error {
	// Load the CA certificate
	caCert, err := os.ReadFile(caCertPath)
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
	jwtToken, err := os.ReadFile(accessTokenPath)
	if err != nil {
		return fmt.Errorf("failed to read JWT token from file: %v", err)
	}
	// Convert JWT token to string and trim whitespace
	tokenString := strings.TrimSpace(string(jwtToken))

	target := fmt.Sprintf("%s:%d", address, port)
	conn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(
			creds,
		),
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
				MacId:     mac,
				SutIp:     ip,
				Uuid:      uuid,
				Serialnum: serial,
			},
		},
	}
	// Create a NodeRequest object and set the Payload field
	nodeRequest := &pb.CreateNodesRequest{
		Payload: []*pb.NodeData{nodeData},
	}
	// Call the gRPC endpoint with the NodeRequest
	var nodeResponse *pb.CreateNodesResponse
	nodeResponse, err = cli.CreateNodes(ctx, nodeRequest)
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

// RetryInfraOnboardNode retries the interactive onboarding process with exponential backoff.
func RetryInfraOnboardNode(ctx context.Context, obmSVC string, obmPort int, macAddr string, ipAddress string, uuid string, serialNumber string, caCertPath string, accessTokenFile string) error {
	maxRetries := 3
	retryDelay := 2 * time.Second // Fixed delay between retries

	for retries := 0; retries < maxRetries; retries++ {
		err := GrpcInfraOnboardNodeJWT(ctx, obmSVC, obmPort, macAddr, ipAddress, uuid, serialNumber, caCertPath, accessTokenFile)
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
