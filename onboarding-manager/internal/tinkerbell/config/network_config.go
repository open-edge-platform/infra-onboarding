// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"gopkg.in/yaml.v3"
)

var (
	configLogger = logging.GetLogger("NetworkConfig")
)

// NetworkConfig represents the complete network configuration
type NetworkConfig struct {
	Interfaces []InterfaceConfig `yaml:"interfaces" json:"interfaces"`
	VLANs      []VLANConfig      `yaml:"vlans,omitempty" json:"vlans,omitempty"`
	Bonds      []BondConfig      `yaml:"bonds,omitempty" json:"bonds,omitempty"`
	Routes     []RouteConfig     `yaml:"routes,omitempty" json:"routes,omitempty"`
}

// InterfaceConfig represents an ethernet interface configuration
type InterfaceConfig struct {
	Name       string   `yaml:"name" json:"name"`
	MacAddress string   `yaml:"mac_address" json:"mac_address"`
	Addresses  []string `yaml:"addresses,omitempty" json:"addresses,omitempty"`
	Gateway    string   `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	DHCPMode   bool     `yaml:"dhcp,omitempty" json:"dhcp,omitempty"`
	DNS        []string `yaml:"dns,omitempty" json:"dns,omitempty"`
	MTU        int      `yaml:"mtu,omitempty" json:"mtu,omitempty"`
	Optional   bool     `yaml:"optional,omitempty" json:"optional,omitempty"`
}

// VLANConfig represents a VLAN configuration
type VLANConfig struct {
	Name      string   `yaml:"name" json:"name"`
	ID        int      `yaml:"id" json:"id"`
	Link      string   `yaml:"link" json:"link"`
	Addresses []string `yaml:"addresses,omitempty" json:"addresses,omitempty"`
	Gateway   string   `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	DNS       []string `yaml:"dns,omitempty" json:"dns,omitempty"`
}

// BondConfig represents a bond configuration
type BondConfig struct {
	Name       string   `yaml:"name" json:"name"`
	Interfaces []string `yaml:"interfaces" json:"interfaces"`
	Mode       string   `yaml:"mode" json:"mode"`
	Addresses  []string `yaml:"addresses,omitempty" json:"addresses,omitempty"`
	Gateway    string   `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	DNS        []string `yaml:"dns,omitempty" json:"dns,omitempty"`
}

// RouteConfig represents a custom route configuration
type RouteConfig struct {
	To        string `yaml:"to" json:"to"`
	Via       string `yaml:"via" json:"via"`
	Interface string `yaml:"interface,omitempty" json:"interface,omitempty"`
	Metric    int    `yaml:"metric,omitempty" json:"metric,omitempty"`
}

// NetworkConfigManager handles loading and validation of network configurations
type NetworkConfigManager struct {
	configPaths []string
}

// NewNetworkConfigManager creates a new network configuration manager
func NewNetworkConfigManager(configPaths ...string) *NetworkConfigManager {
	if len(configPaths) == 0 {
		// Default configuration paths
		configPaths = []string{
			"/etc/infra-onboarding/network-config.yaml",
			"/etc/infra-onboarding/network-config.yml",
			"./configs/network-config.yaml",
			"./configs/network-config.yml",
		}
	}
	
	return &NetworkConfigManager{
		configPaths: configPaths,
	}
}

// LoadNetworkConfig loads and validates network configuration from available config files
func (ncm *NetworkConfigManager) LoadNetworkConfig() (*NetworkConfig, error) {
	var configFile string
	var found bool

	// Check for config file existence
	for _, path := range ncm.configPaths {
		if _, err := os.Stat(path); err == nil {
			configFile = path
			found = true
			configLogger.Info().Msgf("Found network config file: %s", path)
			break
		}
	}

	if !found {
		configLogger.Info().Msg("No network config file found, using single NIC configuration")
		return nil, nil
	}

	// Load configuration file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read network config file %s: %w", configFile, err)
	}

	var config NetworkConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse network config file %s: %w", configFile, err)
	}

	// Validate configuration
	if err := ncm.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid network configuration in %s: %w", configFile, err)
	}

	configLogger.Info().Msgf("Successfully loaded and validated network config with %d interfaces", len(config.Interfaces))
	return &config, nil
}

// validateConfig validates the network configuration
func (ncm *NetworkConfigManager) validateConfig(config *NetworkConfig) error {
	if len(config.Interfaces) == 0 {
		return fmt.Errorf("at least one interface must be configured")
	}

	interfaceNames := make(map[string]bool)
	macAddresses := make(map[string]bool)

	// Validate interfaces
	for i, iface := range config.Interfaces {
		if err := ncm.validateInterface(&iface, i); err != nil {
			return err
		}

		// Check for duplicate names
		if interfaceNames[iface.Name] {
			return fmt.Errorf("duplicate interface name: %s", iface.Name)
		}
		interfaceNames[iface.Name] = true

		// Check for duplicate MAC addresses
		if iface.MacAddress != "" {
			if macAddresses[iface.MacAddress] {
				return fmt.Errorf("duplicate MAC address: %s", iface.MacAddress)
			}
			macAddresses[iface.MacAddress] = true
		}
	}

	// Validate VLANs
	for i, vlan := range config.VLANs {
		if err := ncm.validateVLAN(&vlan, i, interfaceNames); err != nil {
			return err
		}
	}

	// Validate bonds
	for i, bond := range config.Bonds {
		if err := ncm.validateBond(&bond, i, interfaceNames); err != nil {
			return err
		}
	}

	// Validate routes
	for i, route := range config.Routes {
		if err := ncm.validateRoute(&route, i, interfaceNames); err != nil {
			return err
		}
	}

	return nil
}

// validateInterface validates an interface configuration
func (ncm *NetworkConfigManager) validateInterface(iface *InterfaceConfig, index int) error {
	if iface.Name == "" {
		return fmt.Errorf("interface[%d]: name is required", index)
	}

	if iface.MacAddress == "" {
		return fmt.Errorf("interface[%d] (%s): mac_address is required", index, iface.Name)
	}

	// Validate MAC address format
	if !isValidMACAddress(iface.MacAddress) {
		return fmt.Errorf("interface[%d] (%s): invalid MAC address format: %s", index, iface.Name, iface.MacAddress)
	}

	// If not DHCP mode, validate addresses
	if !iface.DHCPMode && len(iface.Addresses) == 0 {
		return fmt.Errorf("interface[%d] (%s): addresses required when dhcp is false", index, iface.Name)
	}

	// Validate IP addresses
	for _, addr := range iface.Addresses {
		if !isValidCIDR(addr) {
			return fmt.Errorf("interface[%d] (%s): invalid IP address/CIDR: %s", index, iface.Name, addr)
		}
	}

	// Validate gateway
	if iface.Gateway != "" && !isValidIP(iface.Gateway) {
		return fmt.Errorf("interface[%d] (%s): invalid gateway IP: %s", index, iface.Name, iface.Gateway)
	}

	return nil
}

// validateVLAN validates a VLAN configuration
func (ncm *NetworkConfigManager) validateVLAN(vlan *VLANConfig, index int, interfaceNames map[string]bool) error {
	if vlan.Name == "" {
		return fmt.Errorf("vlan[%d]: name is required", index)
	}

	if vlan.ID < 1 || vlan.ID > 4094 {
		return fmt.Errorf("vlan[%d] (%s): VLAN ID must be between 1 and 4094", index, vlan.Name)
	}

	if vlan.Link == "" {
		return fmt.Errorf("vlan[%d] (%s): link interface is required", index, vlan.Name)
	}

	if !interfaceNames[vlan.Link] {
		return fmt.Errorf("vlan[%d] (%s): link interface '%s' not found in interfaces", index, vlan.Name, vlan.Link)
	}

	// Validate IP addresses
	for _, addr := range vlan.Addresses {
		if !isValidCIDR(addr) {
			return fmt.Errorf("vlan[%d] (%s): invalid IP address/CIDR: %s", index, vlan.Name, addr)
		}
	}

	return nil
}

// validateBond validates a bond configuration
func (ncm *NetworkConfigManager) validateBond(bond *BondConfig, index int, interfaceNames map[string]bool) error {
	if bond.Name == "" {
		return fmt.Errorf("bond[%d]: name is required", index)
	}

	if len(bond.Interfaces) < 2 {
		return fmt.Errorf("bond[%d] (%s): at least 2 interfaces required for bonding", index, bond.Name)
	}

	validModes := []string{"balance-rr", "active-backup", "balance-xor", "broadcast", "802.3ad", "balance-tlb", "balance-alb"}
	if bond.Mode == "" {
		bond.Mode = "active-backup" // Default mode
	} else {
		found := false
		for _, mode := range validModes {
			if bond.Mode == mode {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("bond[%d] (%s): invalid bond mode '%s', valid modes: %s", index, bond.Name, bond.Mode, strings.Join(validModes, ", "))
		}
	}

	// Validate member interfaces exist
	for _, memberIface := range bond.Interfaces {
		if !interfaceNames[memberIface] {
			return fmt.Errorf("bond[%d] (%s): member interface '%s' not found in interfaces", index, bond.Name, memberIface)
		}
	}

	return nil
}

// validateRoute validates a route configuration
func (ncm *NetworkConfigManager) validateRoute(route *RouteConfig, index int, interfaceNames map[string]bool) error {
	if route.To == "" {
		return fmt.Errorf("route[%d]: destination (to) is required", index)
	}

	if route.Via == "" && route.Interface == "" {
		return fmt.Errorf("route[%d]: either via (gateway) or interface is required", index)
	}

	if route.Interface != "" && !interfaceNames[route.Interface] {
		return fmt.Errorf("route[%d]: interface '%s' not found in interfaces", index, route.Interface)
	}

	return nil
}

// GetConfigSearchPaths returns the search paths for configuration files
func (ncm *NetworkConfigManager) GetConfigSearchPaths() []string {
	return ncm.configPaths
}

// Helper functions for validation
func isValidMACAddress(mac string) bool {
	// Basic MAC address validation (accepts formats like 00:11:22:33:44:55 or 00-11-22-33-44-55)
	if len(mac) != 17 {
		return false
	}
	
	parts := strings.FieldsFunc(mac, func(c rune) bool {
		return c == ':' || c == '-'
	})
	
	if len(parts) != 6 {
		return false
	}
	
	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
		for _, char := range part {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	
	return true
}

func isValidCIDR(cidr string) bool {
	// Basic CIDR validation - checks for IP/prefix format
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return false
	}
	
	return isValidIP(parts[0]) && isValidPrefix(parts[1])
}

func isValidIP(ip string) bool {
	// Basic IP validation
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		
		num := 0
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
			num = num*10 + int(char-'0')
		}
		
		if num > 255 {
			return false
		}
	}
	
	return true
}

func isValidPrefix(prefix string) bool {
	num := 0
	for _, char := range prefix {
		if char < '0' || char > '9' {
			return false
		}
		num = num*10 + int(char-'0')
	}
	
	return num >= 0 && num <= 32
}


