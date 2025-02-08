//go:build mage
// +build mage

/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Test mg.Namespace

// Runs Go tests.
func (Test) Go() error {
	os.Setenv("HTTP_PROXY", "http://proxy-dmz.intel.com:912")
	os.Setenv("HTTPS_PROXY", "http://proxy-dmz.intel.com:912")
	os.Setenv("http_proxy", "http://proxy-dmz.intel.com:912")
	os.Setenv("https_proxy", "http://proxy-dmz.intel.com:912")
	os.Setenv("NO_PROXY", "localhost,127.0.0.1,.intel.com,10.49.76.106")
	os.Setenv("no_proxy", "localhost,127.0.0.1,.intel.com,10.49.76.106")
	// NOTE: Requires ginkgo v2 binary
	// TODO: Reintroduce -race detection once figuring out CGO with musl
	return sh.RunV("ginkgo", "--randomize-all", "--randomize-suites", "-v", "-r", "-tags", "unit", "--cover", "--coverprofile=.coverage-report.out", ".")
}
