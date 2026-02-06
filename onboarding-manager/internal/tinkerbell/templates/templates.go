// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package templates provides functionality for onboarding management.
package templates

import (
	_ "embed"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
)

// MicrovisorTemplate defines a configuration value.
//
//go:embed microvisor.yaml
var MicrovisorTemplate []byte

// MicrovisorName defines a configuration value.
var MicrovisorName = "microvisor"

// UbuntuTemplate defines a configuration value.
//
//go:embed ubuntu.yaml
var UbuntuTemplate []byte

// UbuntuTemplateName defines a configuration value.
var UbuntuTemplateName = "ubuntu"

// TemplatesMap defines a configuration value.
var TemplatesMap = map[string][]byte{
	MicrovisorName:     MicrovisorTemplate,
	UbuntuTemplateName: UbuntuTemplate,
}

// OSTypeToTemplateName defines a configuration value.
// TODO: This uses OS type now but should be based on OS distro or profile name.
var OSTypeToTemplateName = map[osv1.OsType]string{
	osv1.OsType_OS_TYPE_MUTABLE:   UbuntuTemplateName,
	osv1.OsType_OS_TYPE_IMMUTABLE: MicrovisorName,
}
