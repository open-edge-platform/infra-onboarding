// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package vpro

import (
	_ "embed"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/curation"
)

//go:embed Installer.tmpl
var installerTemplate string

// CurateVProInstaller curates the vPro installer script with infra configuration
func CurateVProInstaller(infraConfig config.InfraConfig, osType osv1.OsType) (string, error) {
	// Get all template variables from common infra configuration
	templateVars, err := curation.GetCommonInfraTemplateVariables(infraConfig, osType)
	if err != nil {
		return "", err
	}

	// Curate the template with variables
	curatedScript, err := curation.CurateFromTemplate(installerTemplate, templateVars)
	if err != nil {
		return "", err
	}

	return curatedScript, nil
}
