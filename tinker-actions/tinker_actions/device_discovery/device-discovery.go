// #####################################################################################
// # INTEL CONFIDENTIAL                                                                #
// # Copyright (C) 2024 Intel Corporation                                              #
// # This software and the related documents are Intel copyrighted materials,          #
// # and your use of them is governed by the express license under which they          #
// # were provided to you ("License"). Unless the License provides otherwise,          #
// # you may not use, modify, copy, publish, distribute, disclose or transmit          #
// # this software or the related documents without Intel's prior written permission.  #
// # This software and the related documents are provided as is, with no express       #
// # or implied warranties, other than those that are expressly stated in the License. #
// #####################################################################################

package main

import (
	"bufio"
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	pb_om "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/pkg/api"
	"golang.org/x/oauth2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

const (
	tokenFolder             = "/dev/shm"
	envConfigPath           = "/etc/hook/env_config"
	extraHostsFile          = "/etc/hosts"
	accessTokenFile         = tokenFolder + "/idp_access_token"
	releaseTokenFile        = tokenFolder + "/release_token"
	keycloakTokenURL        = "/realms/master/protocol/openid-connect/token"
	releaseTokenURL         = "/token"
	clientCredentialsFolder = "/dev/shm/"
	clientIDPath            = clientCredentialsFolder + "/client_id"
	clientSecretPath        = clientCredentialsFolder + "/client_secret"
	kernelArgsFilePath      = "/host_proc_cmdline"
	caCertPath              = "/usr/local/share/ca-certificates/ca.crt"
	projectIDPath           = clientCredentialsFolder + "/project_id"
)

func updateHosts(extraHosts string) error {
	// Update hosts if they were provided
	if extraHosts != "" {
		// Replace commas with newlines and remove double quotes
		extraHostsNeeded := strings.ReplaceAll(extraHosts, ",", "\n")
		extraHostsNeeded = strings.ReplaceAll(extraHostsNeeded, "\"", "")

		// Append to /etc/hosts
		hostsFile := "/etc/hosts"
		err := os.WriteFile(hostsFile, []byte(extraHostsNeeded), os.ModeAppend|0644)
		if err != nil {
			return fmt.Errorf("error updating /etc/hosts: %w", err)
		}

		fmt.Println("Adding extra host mappings completed")
	}
	return nil
}

func loadEnvConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && line[0] != '#' {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}
	return scanner.Err()
}

// saveToFile writes data to the specified file path with the given permissions
func saveToFile(path, data string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use io.Writer interface to write data
	_, err = io.WriteString(file, data)
	return err
}

// readEnvVars checks if all required environment variables are set and returns an error if any are missing.
func readEnvVars(requiredVars []string, optionalVars []string) (map[string]string, error) {
	envVars := make(map[string]string)

	// Process required environment variables
	for _, key := range requiredVars {
		value, exists := os.LookupEnv(key)
		if !exists || value == "" {
			return nil, fmt.Errorf("environment variable %s is missing", key)
		}
		envVars[key] = value
	}

	// Process optional environment variables
	for _, key := range optionalVars {
		value, exists := os.LookupEnv(key)
		if !exists || value == "" {
			continue // Skip if the optional variable doesn't exist or is empty
		}
		envVars[key] = value
	}

	return envVars, nil
}

func grpcMaestroOnboardNodeJWT(ctx context.Context, address string, port int, mac string, ip string, uuid string, serial string, caCertPath string, accessTokenPath string) error {
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

	cli := pb_om.NewNodeArtifactServiceNBClient(conn)
	// Create a NodeData object
	nodeData := &pb_om.NodeData{
		Hwdata: []*pb_om.HwData{
			{
				MacId:        mac,
				SutIp:        ip,
				Uuid:         uuid,
				Serialnum:    serial,
				BmcInterface: false,
			},
		},
	}
	// Create a NodeRequest object and set the Payload field
	nodeRequest := &pb_om.NodeRequest{
		Payload: []*pb_om.NodeData{nodeData},
	}
	// Call the gRPC endpoint with the NodeRequest
	var nodeResponse *pb_om.NodeResponse
	nodeResponse, err = cli.CreateNodes(ctx, nodeRequest)
	if err != nil {
		return fmt.Errorf("could not call gRPC endpoint for server %s: %v", target, err)
	}

	// Check if the ProjectId field is empty
	if nodeResponse.ProjectId == "" {
		return fmt.Errorf("received empty Project ID")
	}

	// Save the Project ID to a file
	if err := saveToFile(projectIDPath, nodeResponse.ProjectId); err != nil {
		return fmt.Errorf("failed to save Project ID to file: %v", err)
	}
	return nil
}

func deviceDiscovery(debug bool, timeout time.Duration, obsSVC string, obmSVC string, obmPort int, keycloakURL string, macAddr string, uuid string, serialNumber string, ipAddress string, caCertPath string) {
	if debug {
		// Set a timeout when debug is true
		ctx, cancel := context.WithTimeout(context.Background(), timeout) // Set the timeout you want
		defer cancel()

		fmt.Println("Starting gRPC client with timeout")
		grpcClient(ctx, obsSVC, obmSVC, obmPort, keycloakURL, macAddr, uuid, serialNumber, ipAddress, caCertPath)
	} else {
		// Run without timeout if debug is false
		fmt.Println("Starting gRPC client without timeout")
		grpcClient(context.Background(), obsSVC, obmSVC, obmPort, keycloakURL, macAddr, uuid, serialNumber, ipAddress, caCertPath)
	}
}

func grpcClient(ctx context.Context, obsSVC string, obmSVC string, obmPort int, keycloakURL string, macAddr string, uuid string, serialNumber string, ipAddress string, caCertPath string) {
	// grpc streaming starts here
	// time.Sleep(time.Second * 20)
	clientID, clientSecret, err, fallback := grpcStreamClient(ctx, obsSVC, obmPort, macAddr, uuid, serialNumber, ipAddress, caCertPath)
	if fallback {
		fmt.Printf("Executing fallback method because of error: %s\n", err)
		// Interactive client Auth starts here
		cmd := exec.CommandContext(ctx, "/bin/sh", "client-auth.sh")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := cmd.Start(); err != nil {
			fmt.Println("Error starting command:", err)
			return
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			if err != nil {
				fmt.Println("Error executing command:", err)
				os.Exit(1)
			} else {
				fmt.Println("client-auth.sh executed successfully")
			}
		case <-ctx.Done():
			fmt.Println("client-auth.sh timed out, killing process group...")
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) // Kill the process group
			os.Exit(1)
		}
		// Interactive client Auth ends here

		maxRetries := 3
		retryDelay := 2 * time.Second // Fixed delay between retries

		for retries := 0; retries < maxRetries; retries++ {
			err := grpcMaestroOnboardNodeJWT(ctx, obmSVC, obmPort, macAddr, ipAddress, uuid, serialNumber, caCertPath, accessTokenFile)
			if err == nil {
				fmt.Println("Device discovery done")
				return
			}

			// Log the error and retry info
			fmt.Printf("There was an error in updating the edge-node details with the onboarding manager: %v\n", err)
			if retries < maxRetries-1 {
				fmt.Printf("Retrying update... attempt %d of %d\n", retries+2, maxRetries) // retries+2 to show next attempt
				time.Sleep(retryDelay + time.Duration(rand.Intn(1000))*time.Millisecond)   // Slight random jitter
			}
		}
		// If we exhausted the retries
		log.Fatalf("Max retries reached. Could not complete device discovery.")
	} else {
		if err != nil {
			log.Fatalf("Error Case: %v", err)
		}
		if err := saveToFile(clientIDPath, clientID); err != nil {
			log.Fatalf("error writing clientID: %v", err)
		}
		if err := saveToFile(clientSecretPath, clientSecret); err != nil {
			log.Fatalf("error writing clientSecret: %v", err)
		}
		fmt.Println("Credentials written successfully.")
		// grpc streaming ends here

		// client auth starts here
		idpAccessToken, releaseToken, err := clientAuth(clientID, clientSecret, keycloakURL, keycloakTokenURL, releaseTokenURL, caCertPath)
		if err != nil {
			log.Fatalf("Error: %v\n", err)
		}

		// Write access_token to idp_access_token file
		if err := saveToFile(accessTokenFile, idpAccessToken); err != nil {
			log.Fatalf("failed to save access token to file: %v", err)
		}
		// Write release_token to release_token file
		if err := saveToFile(releaseTokenFile, releaseToken); err != nil {
			log.Fatalf("failed to save release token to file: %v", err)
		}
		// client auth ends here
	}
}

func main() {
	// Define the required environment variables
	requiredVars := []string{
		"onboarding_manager_svc",
		"onboarding_stream_svc",
		"OBM_PORT",
		"KEYCLOAK_URL",
	}

	optionalVars := []string{
		"EXTRA_HOSTS",
	}

	// Load environment variables from env_config
	if err := loadEnvConfig(envConfigPath); err != nil {
		log.Fatalf("Failed to load env_config: %v", err)
	}

	// Check and load the environment variables
	envVars, err := readEnvVars(requiredVars, optionalVars)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Environment variables from the envVars map
	fmt.Println("Environment variables loaded successfully:", envVars)

	obmPort, err := strconv.Atoi(envVars["OBM_PORT"])
	if err != nil {
		log.Fatalf("Error converting port to integer: %v\n", err)

	}

	// parse kernel args
	cfg, err := parseKernelArguments(kernelArgsFilePath)
	if err != nil {
		log.Fatalf("Error parsing kernel arguments: %v\n", err)
	}
	// Use cfg as needed, for example, printing the parsed configuration
	macAddr := cfg.workerID

	// Convert string to bool
	debug, err := strconv.ParseBool(cfg.debug)
	if err != nil {
		fmt.Println("Error parsing DEBUG:", err)
		return
	}

	// Convert string to time.Duration
	timeout, err := time.ParseDuration(cfg.timeout)
	if err != nil {
		fmt.Println("Error parsing TIMEOUT:", err)
		return
	}

	// add extra hosts
	extraHosts, exists := envVars["EXTRA_HOSTS"]
	if exists && extraHosts != "" {
		if err := updateHosts(extraHosts); err != nil {
			log.Fatalf("Failed to add extra hosts: %v", err)
		}
	} else {
		log.Println("No extra hosts provided, skipping update.")
	}

	// logic to detect serial, uuid, and ip based on mac starts here
	serialNumber, err := getSerialNumber()
	if err != nil {
		log.Fatalf("Error getting serial number: %v\n", err)
	}

	uuid, err := getUUID()
	if err != nil {
		log.Fatalf("Error getting UUID: %v\n", err)
	}
	ipAddress, err := getIPAddress(macAddr)
	if err != nil {
		log.Fatal("Error getting IP address: ", err)
	}
	// logic to detect serial, uuid, and ip based on mac ends here

	deviceDiscovery(debug, timeout, envVars["onboarding_stream_svc"], envVars["onboarding_manager_svc"], obmPort, envVars["KEYCLOAK_URL"], macAddr, uuid, serialNumber, ipAddress, caCertPath)
}
