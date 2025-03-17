// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	_ "embed"
	"strings"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/config"
	"github.com/open-edge-platform/infra-onboarding/dkam/pkg/curation"
)

var (
	//go:embed infra.cfg
	cloudInitTemplate string

	zlog = logging.GetLogger("CloudInitGenerator")
)

func templateVariablesFromOptions(options cloudInitOptions) map[string]interface{} {
	extraVars := make(map[string]interface{}, 0)

	extraVars["DEV_MODE"] = false
	if options.useDevMode {
		extraVars["DEV_MODE"] = true
		extraVars["DEV_USER"] = options.devUsername
		extraVars["DEV_USER_PASSWD"] = options.devUserPasswd
	}

	extraVars["TENANT_ID"] = options.tenantID
	extraVars["HOSTNAME"] = options.hostname

	return extraVars
}

func GenerateFromInfraConfig(infraConfig config.InfraConfig, opts ...Option) (string, error) {
	options := defaultCloudInitOptions()
	for _, opt := range opts {
		opt(&options)
	}

	if err := options.validate(); err != nil {
		zlog.Error().Err(err).Msgf("")
		return "", err
	}

	zlog.InfraSec().Info().Msgf("Generating cloud init with options: %+v", options)

	tmplVariables, err := curation.GetCommonInfraTemplateVariables(infraConfig, options.OsType)
	if err != nil {
		return "", err
	}
	tmplVariables["DNS_SERVERS"] = strings.Join(infraConfig.DNSServers, " ")

	extraVars := templateVariablesFromOptions(options)
	for key, value := range extraVars {
		tmplVariables[key] = value
	}

	return curation.CurateFromTemplate(cloudInitTemplate, tmplVariables)
}
