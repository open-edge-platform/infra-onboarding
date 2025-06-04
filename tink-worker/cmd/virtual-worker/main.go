// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/tinkerbell/tink/cmd/virtual-worker/cmd"
)

// version is set at build time.
var version = "devel"

func main() {
	rootCmd := cmd.NewRootCommand(version)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
