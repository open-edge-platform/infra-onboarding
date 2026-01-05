// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sysinfo

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
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
