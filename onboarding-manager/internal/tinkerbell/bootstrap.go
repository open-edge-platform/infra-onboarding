package tinkerbell

import (
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell/templates"
)

func Bootstrap() error {
	zlog.Info().Msg("Bootstrapping Tinkerbell state")

	if err := clearAllTemplates(); err != nil {
		return err
	}

	if err := createTemplates(); err != nil {
		return err
	}

	return nil
}

func createTemplates() error {
	zlog.Info().Msg("Creating pre-defined Tinkerbell templates")
	for name, tmplData := range templates.TemplatesMap {
		template := NewTemplate(string(tmplData), name, env.K8sNamespace)
		if err := CreateTemplate(template); err != nil {
			return err
		}
	}

	return nil
}

func clearAllTemplates() error {
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
