// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// getSerialNumber retrieves the serial number of the machine.
func getSerialNumber() (string, error) {
	cmd := exec.Command("/usr/sbin/dmidecode", "-s", "system-serial-number")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get serial number: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// getUUID retrieves the UUID of the machine.
func getUUID() (string, error) {
	cmd := exec.Command("/usr/sbin/dmidecode", "-s", "system-uuid")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get UUID: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// getIPAddress retrieves the IP address associated with a given MAC address.
func getIPAddress(macAddr string) (string, error) {
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
