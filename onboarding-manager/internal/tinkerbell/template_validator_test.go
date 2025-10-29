// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell/templates"
)

func TestValidateTemplateYAML(t *testing.T) {
	t.Run("valid ubuntu template", func(t *testing.T) {
		err := ValidateTemplateYAML(templates.UbuntuTemplateName)
		assert.NoError(t, err, "Ubuntu template should be valid")
	})

	t.Run("valid microvisor template", func(t *testing.T) {
		err := ValidateTemplateYAML(templates.MicrovisorName)
		assert.NoError(t, err, "Microvisor template should be valid")
	})

	t.Run("non-existent template", func(t *testing.T) {
		err := ValidateTemplateYAML("non-existent-template")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in templates map")
	})

	t.Run("empty template name", func(t *testing.T) {
		err := ValidateTemplateYAML("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in templates map")
	})
}

func TestValidateAllTemplates(t *testing.T) {
	t.Run("all templates are valid", func(t *testing.T) {
		err := ValidateAllTemplates()
		assert.NoError(t, err, "All templates should be valid YAML")
	})
}

func TestIsValidTemplateName(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		expected     bool
	}{
		{
			name:         "ubuntu template exists",
			templateName: templates.UbuntuTemplateName,
			expected:     true,
		},
		{
			name:         "microvisor template exists",
			templateName: templates.MicrovisorName,
			expected:     true,
		},
		{
			name:         "non-existent template",
			templateName: "non-existent",
			expected:     false,
		},
		{
			name:         "empty template name",
			templateName: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidTemplateName(tt.templateName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateStructureValidation(t *testing.T) {
	t.Run("templates have required fields", func(t *testing.T) {
		for templateName := range templates.TemplatesMap {
			err := ValidateTemplateYAML(templateName)
			require.NoError(t, err, "Template %s should have all required fields", templateName)
		}
	})

	t.Run("templates have tasks", func(t *testing.T) {
		for templateName := range templates.TemplatesMap {
			err := ValidateTemplateYAML(templateName)
			require.NoError(t, err, "Template %s should have tasks defined", templateName)
		}
	})
}

func TestTemplateYAMLFormat(t *testing.T) {
	t.Run("templates are parseable YAML", func(t *testing.T) {
		for templateName, templateContent := range templates.TemplatesMap {
			assert.NotEmpty(t, templateContent, "Template %s should not be empty", templateName)

			err := ValidateTemplateYAML(templateName)
			assert.NoError(t, err, "Template %s should be valid YAML", templateName)
		}
	})
}
