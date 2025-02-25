// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	_ "embed"

	"github.com/intel/infra-onboarding/dkam/pkg/config"
	"github.com/intel/infra-onboarding/dkam/pkg/curation"
)

//go:embed infra.cfg
var cloudInitTemplate string

func GenerateFromInfraConfig(options CloudInitOptions) (string, error) {
	tmplVariables, err := curation.GetCommonInfraTemplateVariables(config.GetInfraConfig(), options.OsType)
	if err != nil {
		return "", err
	}

	tmplVariables["MODE"] = options.Mode

	return curation.CurateFromTemplate(cloudInitTemplate, tmplVariables)
}
