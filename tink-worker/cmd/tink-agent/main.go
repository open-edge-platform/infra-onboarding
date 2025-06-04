// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/tinkerbell/tink/internal/cli"
)

func main() {
	if err := cli.NewAgent().Execute(); err != nil {
		os.Exit(-1)
	}
}
