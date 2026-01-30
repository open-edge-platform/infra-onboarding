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
	//go:embed 99_infra.cfg
	cloudInitTemplate string

	zlog = logging.GetLogger("CloudInitGenerator")
)

func templateVariablesFromOptions(options cloudInitOptions) map[string]interface{} {
	extraVars := make(map[string]interface{}, 0)

	extraVars["RUN_AS_STANDALONE"] = options.RunAsStandalone

	extraVars["DEV_MODE"] = false
	extraVars["LOCAL_ACCOUNT_ENABLED"] = false
	if options.useDevMode {
		extraVars["DEV_MODE"] = true
		extraVars["DEV_USER"] = options.devUsername
		extraVars["DEV_USER_PASSWD"] = options.devUserPasswd
	}
	if options.useLocalAccount {
		extraVars["LOCAL_ACCOUNT_ENABLED"] = true
		extraVars["LOCAL_ACCOUNT_USERNAME"] = options.localAccountUserName
		extraVars["LOCAL_USER_SSH_KEY"] = options.sshKey
	}

	extraVars["WITH_PRESERVE_IP"] = false
	if options.preserveIP {
		extraVars["WITH_PRESERVE_IP"] = true
		extraVars["HOST_IP"] = options.staticHostIP
	}

	extraVars["TENANT_ID"] = options.tenantID
	extraVars["HOSTNAME"] = options.hostname
	extraVars["HOST_MAC"] = options.hostMAC
	extraVars["CLIENT_ID"] = options.clientID
	extraVars["CLIENT_SECRET"] = options.clientSecret

	return extraVars
}

// GenerateFromInfraConfig performs operations for onboarding management.
func GenerateFromInfraConfig(template string, infraConfig config.InfraConfig, opts ...Option) (string, error) {
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

	if template != "" {
		zlog.InfraSec().Info().Msg("Using custom cloud-init template")
		return curation.CurateFromTemplate(template, tmplVariables)
	}

	return curation.CurateFromTemplate(cloudInitTemplate, tmplVariables)
}
