// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell/templates"
)

const (
	// DummyHardwareName defines a configuration value.
	DummyHardwareName = "eim-dummy-tink-hardware"
)

// Bootstrap performs operations for onboarding management.
func Bootstrap() error {
	zlog.Info().Msg("Bootstrapping Tinkerbell state")

	if err := DeletePredefinedTinkerbellResources(); err != nil {
		return err
	}

	if err := createTemplates(); err != nil {
		return err
	}

	if err := createDummyHardware(); err != nil {
		return err
	}

	return nil
}

func createTemplates() error {
	zlog.Info().Msg("Creating pre-defined Tinkerbell templates")
	for name, tmplData := range templates.TemplatesMap {
		if err := CreateTemplate(env.K8sNamespace, name, tmplData); err != nil {
			return err
		}
	}

	return nil
}

func createDummyHardware() error {
	return CreateHardwareIfNotExists(env.K8sNamespace, DummyHardwareName)
}
