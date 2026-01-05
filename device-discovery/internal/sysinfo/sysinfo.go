// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sysinfo

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// GetSerialNumber retrieves the serial number of the machine.
func GetSerialNumber() (string, error) {
	cmd := exec.Command("/usr/sbin/dmidecode", "-s", "system-serial-number")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get serial number: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// GetUUID retrieves the UUID of the machine.
func GetUUID() (string, error) {
	cmd := exec.Command("/usr/sbin/dmidecode", "-s", "system-uuid")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get UUID: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// GetIPAddress retrieves the IP address associated with a given MAC address.
func GetIPAddress(macAddr string) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if iface.HardwareAddr.String() == macAddr {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", fmt.Errorf("failed to get addresses for interface %s: %w", iface.Name, err)
			}

			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}

				// Skip loopback addresses and only return non-loopback IPs.
				if !ip.IsLoopback() && ip.To4() != nil {
					return ip.String(), nil
				}
			}
		}
	}
	return "", fmt.Errorf("no IP address found for MAC address %s", macAddr)
}

// GetIPAddressWithRetry retrieves the IP address associated with a given MAC address,
// retrying until an IP is assigned or the timeout is reached.
// This is a convenience wrapper around GetIPAddressWithContext with a default timeout.
func GetIPAddressWithRetry(macAddr string, retries int, sleepDuration time.Duration) (string, error) {
	if retries <= 0 {
		retries = 10 // Default from wait_for_ip.sh
	}
	if sleepDuration <= 0 {
		sleepDuration = 3 * time.Second // Default from wait_for_ip.sh
	}

	// Calculate total timeout based on retries and sleep duration
	timeout := time.Duration(retries) * sleepDuration
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return GetIPAddressWithContext(ctx, macAddr, retries, sleepDuration)
}

// GetIPAddressWithContext retrieves the IP address with retry logic and context support.
// This uses a time.Ticker for more efficient periodic checks and respects context cancellation.
// The operation will be cancelled if the context is cancelled or times out before completion.
func GetIPAddressWithContext(ctx context.Context, macAddr string, retries int, sleepDuration time.Duration) (string, error) {
	if retries <= 0 {
		retries = 10
	}
	if sleepDuration <= 0 {
		sleepDuration = 3 * time.Second
	}

	// Try immediately first (attempt 1)
	ip, err := GetIPAddress(macAddr)
	if err == nil && ip != "" {
		fmt.Printf("IP address %s assigned to MAC %s (attempt 1/%d)\n", ip, macAddr, retries)
		return ip, nil
	}

	// If first attempt fails and we only have 1 retry, return error
	if retries == 1 {
		return "", fmt.Errorf("no IP address assigned to MAC %s after %d attempt", macAddr, retries)
	}

	// Use ticker for subsequent attempts (more idiomatic than time.Sleep in a loop)
	ticker := time.NewTicker(sleepDuration)
	defer ticker.Stop()

	attempt := 2 // We already tried once above
	for {
		select {
		case <-ctx.Done():
			// Context cancelled or timed out
			return "", fmt.Errorf("operation cancelled after %d attempts: %w", attempt-1, ctx.Err())

		case <-ticker.C:
			// Periodic check
			ip, err := GetIPAddress(macAddr)
			if err == nil && ip != "" {
				fmt.Printf("IP address %s assigned to MAC %s (attempt %d/%d)\n", ip, macAddr, attempt, retries)
				return ip, nil
			}

			fmt.Printf("Attempt %d/%d: No IP address assigned to MAC %s yet, waiting %v...\n",
				attempt, retries, macAddr, sleepDuration)

			attempt++
			if attempt > retries {
				return "", fmt.Errorf("no IP address assigned to MAC %s after %d attempts", macAddr, retries)
			}
		}
	}
}

// GetPrimaryMAC retrieves the MAC address of the first non-loopback network interface.
func GetPrimaryMAC() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// Skip loopback and interfaces without hardware address
		if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}

		// Check if interface has an IP address
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		hasIP := false
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				hasIP = true
				break
			}
		}

		// Return the MAC address of the first interface with a valid IP
		if hasIP {
			return iface.HardwareAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}
