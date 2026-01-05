// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"device-discovery/internal/auth"
	"device-discovery/internal/client"
	"device-discovery/internal/config"
	"device-discovery/internal/sysinfo"
)

//go:embed ../../client-auth.sh
var ioOnboardingScript []byte

// CLIConfig holds all command-line configuration
type CLIConfig struct {
	// Service endpoints
	ObmSvc       string
	ObsSvc       string
	ObmPort      int
	KeycloakURL  string
	
	// Device information
	MacAddr      string
	SerialNumber string
	UUID         string
	IPAddress    string
	
	// Optional configuration
	ExtraHosts   string
	CaCertPath   string
	Debug        bool
	Timeout      time.Duration
	
	// Auto-detection flags
	AutoDetect   bool
}

func main() {
	cfg := parseCLIFlags()
	
	// Validate required flags if not auto-detecting
	if !cfg.AutoDetect {
		validateConfig(cfg)
	}
	
	// Auto-detect system information if requested
	if cfg.AutoDetect || cfg.MacAddr != "" {
		if err := autoDetectSystemInfo(cfg); err != nil {
			log.Fatalf("Failed to auto-detect system information: %v", err)
		}
	}
	
	// Validate after auto-detection
	validateConfig(cfg)
	
	// Add extra hosts if provided
	if cfg.ExtraHosts != "" {
		if err := config.UpdateHosts(cfg.ExtraHosts); err != nil {
			log.Fatalf("Failed to add extra hosts: %v", err)
		}
	}
	
	// Display configuration
	fmt.Println("Device Discovery Configuration:")
	fmt.Printf("  Onboarding Manager: %s:%d\n", cfg.ObmSvc, cfg.ObmPort)
	fmt.Printf("  Onboarding Stream: %s:%d\n", cfg.ObsSvc, cfg.ObmPort)
	fmt.Printf("  Keycloak URL: %s\n", cfg.KeycloakURL)
	fmt.Printf("  MAC Address: %s\n", cfg.MacAddr)
	fmt.Printf("  Serial Number: %s\n", cfg.SerialNumber)
	fmt.Printf("  UUID: %s\n", cfg.UUID)
	fmt.Printf("  IP Address: %s\n", cfg.IPAddress)
	fmt.Printf("  Debug Mode: %v\n", cfg.Debug)
	if cfg.Debug {
		fmt.Printf("  Timeout: %v\n", cfg.Timeout)
	}
	fmt.Println()
	
	// Run device discovery
	if err := deviceDiscovery(cfg); err != nil {
		log.Fatalf("Device discovery failed: %v", err)
	}
	
	fmt.Println("Device discovery completed successfully")
}

func parseCLIFlags() *CLIConfig {
	cfg := &CLIConfig{}
	
	// Service endpoints
	flag.StringVar(&cfg.ObmSvc, "obm-svc", "", "Onboarding manager service address (required)")
	flag.StringVar(&cfg.ObsSvc, "obs-svc", "", "Onboarding stream service address (required)")
	flag.IntVar(&cfg.ObmPort, "obm-port", 0, "Onboarding manager port (required)")
	flag.StringVar(&cfg.KeycloakURL, "keycloak-url", "", "Keycloak authentication URL (required)")
	
	// Device information
	flag.StringVar(&cfg.MacAddr, "mac", "", "MAC address of the device (required unless auto-detect)")
	flag.StringVar(&cfg.SerialNumber, "serial", "", "Serial number (auto-detected if not provided)")
	flag.StringVar(&cfg.UUID, "uuid", "", "System UUID (auto-detected if not provided)")
	flag.StringVar(&cfg.IPAddress, "ip", "", "IP address (auto-detected from MAC if not provided)")
	
	// Optional configuration
	flag.StringVar(&cfg.ExtraHosts, "extra-hosts", "", "Additional host mappings (comma-separated: 'host1:ip1,host2:ip2')")
	flag.StringVar(&cfg.CaCertPath, "ca-cert", config.CaCertPath, "Path to CA certificate")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug mode with timeout")
	flag.DurationVar(&cfg.Timeout, "timeout", 5*time.Minute, "Timeout duration for debug mode")
	
	// Auto-detection
	flag.BoolVar(&cfg.AutoDetect, "auto-detect", false, "Auto-detect all system information (MAC, serial, UUID, IP)")
	
	// Custom usage message
	flag.Usage = printUsage
	
	flag.Parse()
	
	return cfg
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Device Discovery CLI - Onboard devices to the infrastructure platform\n\n")
	fmt.Fprintf(os.Stderr, "Required Options:\n")
	fmt.Fprintf(os.Stderr, "  -obm-svc string\n")
	fmt.Fprintf(os.Stderr, "        Onboarding manager service address\n")
	fmt.Fprintf(os.Stderr, "  -obs-svc string\n")
	fmt.Fprintf(os.Stderr, "        Onboarding stream service address\n")
	fmt.Fprintf(os.Stderr, "  -obm-port int\n")
	fmt.Fprintf(os.Stderr, "        Onboarding manager port\n")
	fmt.Fprintf(os.Stderr, "  -keycloak-url string\n")
	fmt.Fprintf(os.Stderr, "        Keycloak authentication URL\n")
	fmt.Fprintf(os.Stderr, "  -mac string\n")
	fmt.Fprintf(os.Stderr, "        MAC address of the device (required unless -auto-detect is used)\n")
	fmt.Fprintf(os.Stderr, "\nOptional Device Information:\n")
	fmt.Fprintf(os.Stderr, "  -serial string\n")
	fmt.Fprintf(os.Stderr, "        Serial number (auto-detected if not provided)\n")
	fmt.Fprintf(os.Stderr, "  -uuid string\n")
	fmt.Fprintf(os.Stderr, "        System UUID (auto-detected if not provided)\n")
	fmt.Fprintf(os.Stderr, "  -ip string\n")
	fmt.Fprintf(os.Stderr, "        IP address (auto-detected from MAC if not provided)\n")
	fmt.Fprintf(os.Stderr, "\nAuto-Detection:\n")
	fmt.Fprintf(os.Stderr, "  -auto-detect\n")
	fmt.Fprintf(os.Stderr, "        Auto-detect all system information (MAC, serial, UUID, IP)\n")
	fmt.Fprintf(os.Stderr, "\nAdditional Options:\n")
	fmt.Fprintf(os.Stderr, "  -extra-hosts string\n")
	fmt.Fprintf(os.Stderr, "        Additional host mappings (comma-separated: 'host1:ip1,host2:ip2')\n")
	fmt.Fprintf(os.Stderr, "  -ca-cert string\n")
	fmt.Fprintf(os.Stderr, "        Path to CA certificate (default: %s)\n", config.CaCertPath)
	fmt.Fprintf(os.Stderr, "  -debug\n")
	fmt.Fprintf(os.Stderr, "        Enable debug mode with timeout\n")
	fmt.Fprintf(os.Stderr, "  -timeout duration\n")
	fmt.Fprintf(os.Stderr, "        Timeout duration for debug mode (default: 5m0s)\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  # Auto-detect all system information\n")
	fmt.Fprintf(os.Stderr, "  %s -obm-svc obm.example.com -obs-svc obs.example.com -obm-port 50051 \\\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    -keycloak-url keycloak.example.com -auto-detect\n\n")
	fmt.Fprintf(os.Stderr, "  # Specify MAC address, auto-detect others\n")
	fmt.Fprintf(os.Stderr, "  %s -obm-svc obm.example.com -obs-svc obs.example.com -obm-port 50051 \\\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    -keycloak-url keycloak.example.com -mac 00:11:22:33:44:55\n\n")
	fmt.Fprintf(os.Stderr, "  # Fully manual configuration\n")
	fmt.Fprintf(os.Stderr, "  %s -obm-svc obm.example.com -obs-svc obs.example.com -obm-port 50051 \\\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    -keycloak-url keycloak.example.com -mac 00:11:22:33:44:55 \\\n")
	fmt.Fprintf(os.Stderr, "    -serial ABC123 -uuid 12345678-1234-1234-1234-123456789012 -ip 192.168.1.100\n\n")
	fmt.Fprintf(os.Stderr, "  # With debug mode and extra hosts\n")
	fmt.Fprintf(os.Stderr, "  %s -obm-svc obm.example.com -obs-svc obs.example.com -obm-port 50051 \\\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    -keycloak-url keycloak.example.com -auto-detect -debug -timeout 10m \\\n")
	fmt.Fprintf(os.Stderr, "    -extra-hosts \"registry.local:10.0.0.1,api.local:10.0.0.2\"\n\n")
}

func validateConfig(cfg *CLIConfig) {
	var missing []string
	
	if cfg.ObmSvc == "" {
		missing = append(missing, "-obm-svc")
	}
	if cfg.ObsSvc == "" {
		missing = append(missing, "-obs-svc")
	}
	if cfg.ObmPort == 0 {
		missing = append(missing, "-obm-port")
	}
	if cfg.KeycloakURL == "" {
		missing = append(missing, "-keycloak-url")
	}
	if cfg.MacAddr == "" {
		missing = append(missing, "-mac (or use -auto-detect)")
	}
	if cfg.SerialNumber == "" {
		missing = append(missing, "-serial (or will be auto-detected)")
	}
	if cfg.UUID == "" {
		missing = append(missing, "-uuid (or will be auto-detected)")
	}
	if cfg.IPAddress == "" {
		missing = append(missing, "-ip (or will be auto-detected from MAC)")
	}
	
	// Only fail on critical missing fields
	criticalMissing := []string{}
	for _, field := range missing {
		if !strings.Contains(field, "auto-detect") && 
		   !strings.Contains(field, "-serial") && 
		   !strings.Contains(field, "-uuid") && 
		   !strings.Contains(field, "-ip") {
			criticalMissing = append(criticalMissing, field)
		}
	}
	
	if len(criticalMissing) > 0 {
		fmt.Fprintf(os.Stderr, "Error: Missing required flags: %s\n\n", strings.Join(criticalMissing, ", "))
		flag.Usage()
		os.Exit(1)
	}
}

func autoDetectSystemInfo(cfg *CLIConfig) error {
	var err error
	
	// Auto-detect serial number if not provided
	if cfg.SerialNumber == "" {
		cfg.SerialNumber, err = sysinfo.GetSerialNumber()
		if err != nil {
			return fmt.Errorf("failed to auto-detect serial number: %w", err)
		}
		fmt.Printf("Auto-detected serial number: %s\n", cfg.SerialNumber)
	}
	
	// Auto-detect UUID if not provided
	if cfg.UUID == "" {
		cfg.UUID, err = sysinfo.GetUUID()
		if err != nil {
			return fmt.Errorf("failed to auto-detect UUID: %w", err)
		}
		fmt.Printf("Auto-detected UUID: %s\n", cfg.UUID)
	}
	
	// Auto-detect MAC address if auto-detect flag is set and MAC is empty
	if cfg.AutoDetect && cfg.MacAddr == "" {
		cfg.MacAddr, err = sysinfo.GetPrimaryMAC()
		if err != nil {
			return fmt.Errorf("failed to auto-detect MAC address: %w", err)
		}
		fmt.Printf("Auto-detected MAC address: %s\n", cfg.MacAddr)
	}
	
	// Auto-detect IP address from MAC if not provided
	if cfg.IPAddress == "" && cfg.MacAddr != "" {
		cfg.IPAddress, err = sysinfo.GetIPAddress(cfg.MacAddr)
		if err != nil {
			return fmt.Errorf("failed to auto-detect IP address for MAC %s: %w", cfg.MacAddr, err)
		}
		fmt.Printf("Auto-detected IP address: %s\n", cfg.IPAddress)
	}
	
	return nil
}

func deviceDiscovery(cfg *CLIConfig) error {
	var ctx context.Context
	var cancel context.CancelFunc
	
	if cfg.Debug {
		// Set a timeout when debug is true
		ctx, cancel = context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		fmt.Println("Starting gRPC client with timeout")
	} else {
		// Run without timeout if debug is false
		ctx = context.Background()
		fmt.Println("Starting gRPC client without timeout")
	}
	
	return grpcClient(ctx, cfg)
}

func grpcClient(ctx context.Context, cfg *CLIConfig) error {
	// grpc streaming starts here
	clientID, clientSecret, err, fallback := client.GrpcStreamClient(
		ctx, 
		cfg.ObsSvc, 
		cfg.ObmPort, 
		cfg.MacAddr, 
		cfg.UUID, 
		cfg.SerialNumber, 
		cfg.IPAddress, 
		cfg.CaCertPath,
	)
	
	if fallback {
		fmt.Printf("Executing fallback method because of error: %s\n", err)
		
		// Interactive client Auth starts here
		if err := runClientAuthScript(ctx, ioOnboardingScript); err != nil {
			return fmt.Errorf("failed to run client auth script: %w", err)
		}
		
		// Retry logic for interactive onboarding
		if err := client.RetryInfraOnboardNode(
			ctx, 
			cfg.ObmSvc, 
			cfg.ObmPort, 
			cfg.MacAddr, 
			cfg.IPAddress, 
			cfg.UUID, 
			cfg.SerialNumber, 
			cfg.CaCertPath, 
			config.AccessTokenFile,
		); err != nil {
			return fmt.Errorf("max retries reached, could not complete device discovery: %w", err)
		}
		
		fmt.Println("Device discovery completed (interactive mode)")
		return nil
	}
	
	// Handle non-fallback case
	if err != nil {
		return fmt.Errorf("gRPC stream client error: %w", err)
	}
	
	// Save client credentials
	if err := config.SaveToFile(config.ClientIDPath, clientID); err != nil {
		return fmt.Errorf("failed to save client ID: %w", err)
	}
	
	if err := config.SaveToFile(config.ClientSecretPath, clientSecret); err != nil {
		return fmt.Errorf("failed to save client secret: %w", err)
	}
	
	fmt.Println("Credentials written successfully.")
	
	// Client authentication
	idpAccessToken, releaseToken, err := auth.ClientAuth(
		clientID, 
		clientSecret, 
		cfg.KeycloakURL, 
		config.KeycloakTokenURL, 
		config.ReleaseTokenURL, 
		cfg.CaCertPath,
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
	
	fmt.Println("Tokens saved successfully")
	return nil
}

func runClientAuthScript(ctx context.Context, scriptContent []byte) error {
	tmpfile, err := config.CreateTempScript(scriptContent)
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	}()

	cmd := exec.CommandContext(ctx, "/bin/sh", tmpfile.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				fmt.Printf("STDERR:\n%s\n", string(exitErr.Stderr))
			}
			return fmt.Errorf("error executing command: %w", err)
		}
		fmt.Println("client-auth.sh executed successfully")
		return nil
	case <-ctx.Done():
		fmt.Println("client-auth.sh timed out, killing process group...")
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) // Kill the process group
		return fmt.Errorf("client-auth.sh timed out: %w", ctx.Err())
	}
}
