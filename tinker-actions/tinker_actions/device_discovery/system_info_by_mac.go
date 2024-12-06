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
