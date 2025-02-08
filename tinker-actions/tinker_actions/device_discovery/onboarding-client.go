// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	// pb "stream-test/api"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/pkg/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

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

func grpcStreamClient(ctx context.Context, address string, port int, mac string, uuid string, serial string, ipAddress string, caCertPath string) (string, string, error, bool) {
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
	request := &pb.OnboardStreamRequest{
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
			case pb.OnboardStreamResponse_REGISTERED:
				fmt.Println("Edge node registered. Waiting for the edge node to become ready for onboarding...")

				// Sleep for a randomized backoff duration
				time.Sleep(backoff + time.Duration(rand.Intn(1000))*time.Millisecond)

				// Double the backoff time, but cap it at maxBackoff
				backoff *= 2
				if backoff > maxBackoff {
					backoff = 2 * time.Second
				}

			case pb.OnboardStreamResponse_ONBOARDED:
				clientID := resp.ClientId
				clientSecret := resp.ClientSecret
				projectID := resp.ProjectId

				// Ensure the Project ID is not empty
				if projectID == "" {
					return "", "", fmt.Errorf("received empty Project ID"), fallback
				}

				// Save the Project ID to a file
				if err := saveToFile(projectIDPath, projectID); err != nil {
					return "", "", fmt.Errorf("failed to save Project ID to file: %v", err), fallback
				}

				// Ensure both clientID and clientSecret are not empty
				if clientID == "" || clientSecret == "" {
					return "", "", fmt.Errorf("received empty clientID or clientSecret"), fallback
				}
				return clientID, clientSecret, nil, fallback

			case pb.OnboardStreamResponse_UNSPECIFIED:
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
