// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"
	"strings"
)

type tinkConfig struct {
	workerID string
	debug    string
	timeout  string
}

// parseCmdLine will parse the command line and return either a config or an error.
func parseCmdLine(cmdLines []string) (tinkConfig, error) {
	var cfg tinkConfig
	for i := range cmdLines {
		cmdLine := strings.Split(cmdLines[i], "=")
		if len(cmdLine) == 0 {
			continue
		}

		switch cmd := cmdLine[0]; cmd {
		case "worker_id":
			cfg.workerID = cmdLine[1]
		case "DEBUG":
			cfg.debug = cmdLine[1]
		case "TIMEOUT":
			cfg.timeout = cmdLine[1]
		case "s_net_scan_duration":
			// This is a custom argument for the network scan duration.
			log.Printf("Network scan duration: %s sec", cmdLine[1])
		case "s_vmlinux_download":
			// This is a custom argument for the vmlinux image download duration.
			log.Printf("vmlinux image download duration: %s sec", cmdLine[1])
		case "s_initramfs_download":
			// This is a custom argument for the initramfs image download duration.
			log.Printf("initramfs image download duration: %s sec", cmdLine[1])
		}
	}
	return cfg, nil
}

// parseKernelArguments reads the kernel command line and returns the parsed config or an error.
func parseKernelArguments(kernelArgsFilePath string) (tinkConfig, error) {
	content, err := os.ReadFile(kernelArgsFilePath)
	if err != nil {
		return tinkConfig{}, err
	}
	cmdLines := strings.Split(string(content), " ")
	return parseCmdLine(cmdLines)
}
