// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"os"
	"strings"
)

// TinkConfig holds the parsed kernel configuration.
type TinkConfig struct {
	WorkerID string
	Debug    string
	Timeout  string
}

// parseCmdLine will parse the command line and return either a config or an error.
func parseCmdLine(cmdLines []string) (TinkConfig, error) {
	var cfg TinkConfig
	for i := range cmdLines {
		cmdLine := strings.Split(cmdLines[i], "=")
		if len(cmdLine) == 0 {
			continue
		}

		switch cmd := cmdLine[0]; cmd {
		case "worker_id":
			cfg.WorkerID = cmdLine[1]
		case "DEBUG":
			cfg.Debug = cmdLine[1]
		case "TIMEOUT":
			cfg.Timeout = cmdLine[1]
		}
	}
	return cfg, nil
}

// ParseKernelArguments reads the kernel command line and returns the parsed config or an error.
func ParseKernelArguments(kernelArgsFilePath string) (TinkConfig, error) {
	content, err := os.ReadFile(kernelArgsFilePath)
	if err != nil {
		return TinkConfig{}, err
	}
	cmdLines := strings.Split(string(content), " ")
	return parseCmdLine(cmdLines)
}
