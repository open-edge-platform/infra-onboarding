// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"device-discovery/internal/config"
	"device-discovery/internal/mode"
	"device-discovery/internal/sysinfo"
)

// CLIConfig holds all command-line configuration
type CLIConfig struct {
	// Config file
	ConfigFile string

	// Service endpoints
	ObmSvc      string
	ObsSvc      string
	ObmPort     int
	KeycloakURL string

	// Device information
	MacAddr      string
	SerialNumber string
	UUID         string
	IPAddress    string

	// Optional configuration
	ExtraHosts string
	CaCertPath string
	Debug      bool
	Timeout    time.Duration

	// Auto-detection flags
	AutoDetect bool
}

func main() {
	cfg := parseCLIFlags()

	// Load config file if specified
	if cfg.ConfigFile != "" {
		if err := loadConfigFile(cfg); err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
	}

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

	// Config file
	flag.StringVar(&cfg.ConfigFile, "config", "", "Path to configuration file (optional)")

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
	fmt.Fprintf(os.Stderr, "Device Discovery CLI - Onboard devices to the Open Edge platform\n\n")
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
	fmt.Fprintf(os.Stderr, "\nConfiguration File:\n")
	fmt.Fprintf(os.Stderr, "  -config string\n")
	fmt.Fprintf(os.Stderr, "        Path to configuration file (optional)\n")
	fmt.Fprintf(os.Stderr, "        CLI flags override values from config file\n")
	fmt.Fprintf(os.Stderr, "        Format: KEY=VALUE (one per line, # for comments)\n")
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
	fmt.Fprintf(os.Stderr, "  # Using a configuration file\n")
	fmt.Fprintf(os.Stderr, "  %s -config /path/to/config.env\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  # Override config file with CLI flags\n")
	fmt.Fprintf(os.Stderr, "  %s -config /path/to/config.env -mac 00:11:22:33:44:55 -debug\n\n", os.Args[0])
}

// loadConfigFile loads configuration from a file.
// CLI flags that are explicitly set will override values from the file.
func loadConfigFile(cfg *CLIConfig) error {
	file, err := os.Open(cfg.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Track which flags were explicitly set via CLI
	explicitFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		explicitFlags[f.Name] = true
	})

	// Parse config file
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format at line %d: %s (expected KEY=VALUE)", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		// Apply value only if the corresponding flag wasn't explicitly set
		if err := applyConfigValue(cfg, key, value, explicitFlags); err != nil {
			return fmt.Errorf("error at line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

// applyConfigValue applies a configuration value to the appropriate field.
func applyConfigValue(cfg *CLIConfig, key, value string, explicitFlags map[string]bool) error {
	// Map config file keys to flag names
	keyToFlag := map[string]string{
		"OBM_SVC":      "obm-svc",
		"OBS_SVC":      "obs-svc",
		"OBM_PORT":     "obm-port",
		"KEYCLOAK_URL": "keycloak-url",
		"MAC":          "mac",
		"SERIAL":       "serial",
		"UUID":         "uuid",
		"IP":           "ip",
		"EXTRA_HOSTS":  "extra-hosts",
		"CA_CERT":      "ca-cert",
		"DEBUG":        "debug",
		"TIMEOUT":      "timeout",
		"AUTO_DETECT":  "auto-detect",
	}

	flagName, ok := keyToFlag[key]
	if !ok {
		// Unknown key - skip it with a warning
		fmt.Fprintf(os.Stderr, "Warning: unknown config key '%s' (skipping)\n", key)
		return nil
	}

	// Skip if this flag was explicitly set via CLI
	if explicitFlags[flagName] {
		return nil
	}

	// Apply the value
	switch flagName {
	case "obm-svc":
		cfg.ObmSvc = value
	case "obs-svc":
		cfg.ObsSvc = value
	case "obm-port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid port value '%s': %w", value, err)
		}
		cfg.ObmPort = port
	case "keycloak-url":
		cfg.KeycloakURL = value
	case "mac":
		cfg.MacAddr = value
	case "serial":
		cfg.SerialNumber = value
	case "uuid":
		cfg.UUID = value
	case "ip":
		cfg.IPAddress = value
	case "extra-hosts":
		cfg.ExtraHosts = value
	case "ca-cert":
		cfg.CaCertPath = value
	case "debug":
		debug, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid debug value '%s': %w", value, err)
		}
		cfg.Debug = debug
	case "timeout":
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid timeout value '%s': %w", value, err)
		}
		cfg.Timeout = timeout
	case "auto-detect":
		autoDetect, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid auto-detect value '%s': %w", value, err)
		}
		cfg.AutoDetect = autoDetect
	}

	return nil
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
	// Use retry logic to wait for DHCP assignment if needed
	if cfg.IPAddress == "" && cfg.MacAddr != "" {
		fmt.Printf("Waiting for IP address assignment for MAC %s...\n", cfg.MacAddr)
		cfg.IPAddress, err = sysinfo.GetIPAddressWithRetry(cfg.MacAddr, 10, 3*time.Second)
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
		fmt.Println("Starting device onboarding with timeout")
	} else {
		// Run without timeout if debug is false
		ctx = context.Background()
		fmt.Println("Starting device onboarding without timeout")
	}

	// Create controller configuration
	controllerCfg := mode.Config{
		ObmSvc:       cfg.ObmSvc,
		ObsSvc:       cfg.ObsSvc,
		ObmPort:      cfg.ObmPort,
		KeycloakURL:  cfg.KeycloakURL,
		MacAddr:      cfg.MacAddr,
		SerialNumber: cfg.SerialNumber,
		UUID:         cfg.UUID,
		IPAddress:    cfg.IPAddress,
		CaCertPath:   cfg.CaCertPath,
	}

	// Create and execute onboarding controller
	controller := mode.NewOnboardingController(controllerCfg)
	return controller.Execute(ctx)
}
