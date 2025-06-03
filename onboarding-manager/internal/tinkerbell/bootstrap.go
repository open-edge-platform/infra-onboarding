// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell/templates"
)

const (
	DummyHardwareName = "eim-dummy-tink-hardware"
)

func Bootstrap() error {
	zlog.Info().Msg("Bootstrapping Tinkerbell state")

	if err := clearAllTinkResources(); err != nil {
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

func clearAllTinkResources() error {
	zlog.Info().Msg("Clearing dummy Tinkerbell hardware")
	if err := DeleteHardware(env.K8sNamespace, DummyHardwareName); err != nil {
		return err
	}

	zlog.Info().Msg("Clearing all existing Tinkerbell templates")
	allTemplates, err := ListTemplates()
	if err != nil {
		return err
	}

	for _, tmpl := range allTemplates {
		zlog.Info().Msgf("Deleting Tinkerbell template %q", tmpl.Name)
		if delErr := DeleteTemplate(tmpl.Name, tmpl.Namespace); delErr != nil {
			return delErr
		}
	}

	return nil
}
