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
	"os"
	"strings"
)

type tinkConfig struct {
	workerID string
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
