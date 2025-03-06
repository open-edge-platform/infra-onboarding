// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	_ "embed"

	"github.com/intel/infra-core/inventory/v2/pkg/logging"
	"github.com/intel/infra-onboarding/dkam/pkg/config"
	"github.com/intel/infra-onboarding/dkam/pkg/curation"
)

var (
	//go:embed infra.cfg
	cloudInitTemplate string

	zlog = logging.GetLogger("CloudInitGenerator")
)

func GenerateFromInfraConfig(infraConfig config.InfraConfig, options CloudInitOptions) (string, error) {
	zlog.InfraSec().Info().Msgf("Generating cloud init with options: %+v", options)

	tmplVariables, err := curation.GetCommonInfraTemplateVariables(infraConfig, options.OsType)
	if err != nil {
		return "", err
	}

	tmplVariables["MODE"] = options.Mode

	return curation.CurateFromTemplate(cloudInitTemplate, tmplVariables)
}
