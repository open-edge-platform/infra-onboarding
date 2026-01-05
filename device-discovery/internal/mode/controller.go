// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package mode

import (
	"context"
	"fmt"

	"device-discovery/internal/auth"
	"device-discovery/internal/config"
	"device-discovery/internal/mode/interactive"
	"device-discovery/internal/mode/noninteractive"
)

// OnboardingController manages the device onboarding process, coordinating between
// non-interactive (automatic) and interactive (manual) modes.
type OnboardingController struct {
	// Service endpoints
	obmSvc      string
	obsSvc      string
	obmPort     int
	keycloakURL string

	// Device information
	macAddr      string
	serialNumber string
	uuid         string
	ipAddress    string

	// Configuration
	caCertPath string
}

// Config holds the configuration for the onboarding orchestrator.
type Config struct {
	ObmSvc       string
	ObsSvc       string
	ObmPort      int
	KeycloakURL  string
	MacAddr      string
	SerialNumber string
	UUID         string
	IPAddress    string
	CaCertPath   string
}

// NewOnboardingController creates a new onboarding controller.
func NewOnboardingController(cfg Config) *OnboardingController {
	return &OnboardingController{
		obmSvc:       cfg.ObmSvc,
		obsSvc:       cfg.ObsSvc,
		obmPort:      cfg.ObmPort,
		keycloakURL:  cfg.KeycloakURL,
		macAddr:      cfg.MacAddr,
		serialNumber: cfg.SerialNumber,
		uuid:         cfg.UUID,
		ipAddress:    cfg.IPAddress,
		caCertPath:   cfg.CaCertPath,
	}
}

// Execute performs device onboarding, automatically switching between modes as needed.
// It first attempts non-interactive mode, and falls back to interactive mode if the
// device is not found in the system.
func (o *OnboardingController) Execute(ctx context.Context) error {
	fmt.Println("Starting device onboarding...")

	// Try non-interactive mode first
	result := o.tryNonInteractiveMode(ctx)

	if result.ShouldFallback {
		// Device not found - fall back to interactive mode
		fmt.Printf("Executing fallback to interactive mode because: %v\n", result.Error)
		return o.executeInteractiveMode(ctx)
	}

	if result.Error != nil {
		return fmt.Errorf("non-interactive onboarding failed: %w", result.Error)
	}

	// Non-interactive mode succeeded - complete authentication
	return o.completeNonInteractiveAuth(result.ClientID, result.ClientSecret)
}

// tryNonInteractiveMode attempts automatic onboarding via streaming gRPC.
func (o *OnboardingController) tryNonInteractiveMode(ctx context.Context) noninteractive.StreamResult {
	fmt.Println("Attempting non-interactive (streaming) onboarding...")

	client := noninteractive.NewClient(
		o.obsSvc,
		o.obmPort,
		o.macAddr,
		o.uuid,
		o.serialNumber,
		o.ipAddress,
		o.caCertPath,
	)

	return client.Onboard(ctx)
}

// completeNonInteractiveAuth completes the authentication flow for non-interactive mode.
func (o *OnboardingController) completeNonInteractiveAuth(clientID, clientSecret string) error {
	// Save client credentials
	if err := config.SaveToFile(config.ClientIDPath, clientID); err != nil {
		return fmt.Errorf("failed to save client ID: %w", err)
	}

	if err := config.SaveToFile(config.ClientSecretPath, clientSecret); err != nil {
		return fmt.Errorf("failed to save client secret: %w", err)
	}

	fmt.Println("Credentials written successfully.")

	// Client authentication - exchange credentials for tokens
	idpAccessToken, releaseToken, err := auth.ClientAuth(
		clientID,
		clientSecret,
		o.keycloakURL,
		config.KeycloakTokenURL,
		config.ReleaseTokenURL,
		o.caCertPath,
	)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Write access token
	if err := config.SaveToFile(config.AccessTokenFile, idpAccessToken); err != nil {
		return fmt.Errorf("failed to save access token: %w", err)
	}

	// Write release token
	if err := config.SaveToFile(config.ReleaseTokenFile, releaseToken); err != nil {
		return fmt.Errorf("failed to save release token: %w", err)
	}

	fmt.Println("Tokens saved successfully (non-interactive mode)")
	return nil
}

// executeInteractiveMode performs manual onboarding with user authentication.
func (o *OnboardingController) executeInteractiveMode(ctx context.Context) error {
	fmt.Println("Starting interactive (manual) onboarding...")

	// Step 1: Execute client-auth.sh for TTY-based authentication
	fmt.Println("Executing client authentication script...")
	if err := interactive.ExecuteAuthScript(ctx); err != nil {
		return fmt.Errorf("failed to run client auth script: %w", err)
	}

	// Step 2: Create interactive client with JWT authentication
	client := interactive.NewClient(
		o.obmSvc,
		o.obmPort,
		o.macAddr,
		o.ipAddress,
		o.uuid,
		o.serialNumber,
		o.caCertPath,
		config.AccessTokenFile,
	)

	// Step 3: Perform onboarding with retry logic
	if err := client.OnboardWithRetry(ctx); err != nil {
		return fmt.Errorf("interactive onboarding failed after retries: %w", err)
	}

	fmt.Println("Device discovery completed (interactive mode)")
	return nil
}
