// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// loadCACertPool loads the CA certificate from a file and returns a certificate pool.
func loadCACertPool(caCertPath string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}
	return caCertPool, nil
}

func fetchAccessToken(keycloakURL string, clientID string, clientSecret string, caCertPath string) (string, error) {
	// Prepare the request data
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	reqBody := bytes.NewBufferString(data.Encode())

	// Create the HTTP request
	req, err := http.NewRequest("POST", "https://"+keycloakURL, reqBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Load the CA certificate
	caCertPool, err := loadCACertPool(caCertPath)
	if err != nil {
		return "", fmt.Errorf("error loading CA certificate: %v", err)
	}

	// Create an HTTP client with the CA certificate
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get access token, status: %s", resp.Status)
	}

	// Parse the JSON response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Extract the access token
	token, ok := result["access_token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("access token not found in response")
	}

	return token, nil
}

func fetchReleaseToken(releaseServerURL string, accessToken string, caCertPath string) (string, error) {
	// Ensure the access token is not empty
	if accessToken == "" {
		return "", fmt.Errorf("access token is required")
	}

	// Load CA certificate
	caCertPool, err := loadCACertPool(caCertPath)
	if err != nil {
		return "", fmt.Errorf("error loading CA certificate: %v", err)
	}

	// Construct the HTTP request
	req, err := http.NewRequest("GET", "https://"+releaseServerURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Add the authorization header with the bearer token
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Create an HTTP client with CA certificate
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	switch resp.StatusCode {
	case http.StatusOK:
		// Status 200: Successfully retrieved release token
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("error reading response body: %v", err)
		}
		releaseToken := string(body)
		if releaseToken == "null" || releaseToken == "" {
			return "", fmt.Errorf("invalid token received")
		}
		fmt.Println("Successfully retrieved release token.")
		return releaseToken, nil

	case http.StatusNoContent:
		// Status 204: No release token exists, create an empty file
		fmt.Println("No release token exists. Creating an empty file.")
		return "", nil

	default:
		// Other status codes: Failed to retrieve release token
		return "", fmt.Errorf("failed to retrieve release token. HTTP status code: %d", resp.StatusCode)
	}
}

// clientAuth handles authentication and retrieves tokens
func clientAuth(clientID string, clientSecret string, keycloakURL string, acceskTokenURL string, releaseTokenURL string, caCertPath string) (idpAccessToken string, releaseToken string, err error) {

	// Fetch JWT access token from Keycloak
	idpAccessToken, err = fetchAccessToken(keycloakURL+acceskTokenURL, clientID, clientSecret, caCertPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get JWT access token from Keycloak: %v", err)
	}

	// Fetch release service token
	releaseTokenURL = strings.Replace(keycloakURL, "keycloak", "release", 1) + releaseTokenURL
	releaseToken, err = fetchReleaseToken(releaseTokenURL, idpAccessToken, caCertPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get release service token: %v", err)
	}

	return idpAccessToken, releaseToken, nil
}
