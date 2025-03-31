//go:build mage
// +build mage

/*
 * SPDX-FileCopyrightText: (C) 2025 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Test mg.Namespace

// Runs Go tests.
func (Test) Go() error {
	// NOTE: Requires ginkgo v2 binary
	// TODO: Reintroduce -race detection once figuring out CGO with musl
	return sh.RunV("ginkgo", "--randomize-all", "--randomize-suites", "-v", "-r", "-tags", "unit", "--cover", "--coverprofile=.coverage-report.out", ".")
}
