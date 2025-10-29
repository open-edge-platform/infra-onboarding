// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"bytes"
	"fmt"
	"regexp"
	"text/template"

	"gopkg.in/yaml.v3"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell/templates"
)

// templatePlaceholderRegex matches Go template expressions like {{ .Variable }}
var templatePlaceholderRegex = regexp.MustCompile(`\{\{[^}]+\}\}`)

// ValidateTemplateYAML validates that the template content is proper YAML format with Go templates
func ValidateTemplateYAML(templateName string) error {
	// Get the template content from the templates map
	templateContent, exists := templates.TemplatesMap[templateName]
	if !exists {
		return inv_errors.Errorf("Template '%s' not found in templates map", templateName)
	}

	if len(templateContent) == 0 {
		return inv_errors.Errorf("Template '%s' is empty", templateName)
	}

	// First, validate that it's a valid Go template
	tmpl, err := template.New(templateName).Funcs(template.FuncMap{
		"indent": func(spaces int, text string) string {
			// Simple indent function for template validation
			return text
		},
		"nindent": func(spaces int, text string) string {
			// nindent adds a newline first, then indents each line
			return text
		},
	}).Parse(string(templateContent))
	if err != nil {
		return inv_errors.Errorf("Template '%s' is not a valid Go template: %v", templateName, err)
	}

	// Try to render the template with dummy data to create valid YAML
	// This helps validate the template structure before actual use
	dummyData := createDummyTemplateData()
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, dummyData); err != nil {
		// If template execution fails, it might be due to missing fields
		// We'll try a simpler validation by replacing template variables with placeholders
		return validateTemplateStructure(templateName, templateContent)
	}

	// Parse the rendered YAML to validate format
	var result interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		return inv_errors.Errorf("Template '%s' renders to invalid YAML: %v", templateName, err)
	}

	// Validate Tinkerbell workflow structure
	return validateWorkflowStructure(templateName, result)
}

// validateTemplateStructure validates template by replacing Go template expressions with placeholders
func validateTemplateStructure(templateName string, templateContent []byte) error {
	// Replace all template expressions with dummy values to create parseable YAML
	yamlContent := templatePlaceholderRegex.ReplaceAllString(string(templateContent), "placeholder")

	// Parse the YAML to validate format
	var result interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &result); err != nil {
		return inv_errors.Errorf("Template '%s' structure is not valid YAML: %v", templateName, err)
	}

	return validateWorkflowStructure(templateName, result)
}

// validateWorkflowStructure validates the Tinkerbell workflow structure
func validateWorkflowStructure(templateName string, result interface{}) error {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return inv_errors.Errorf("Template '%s' is not a valid Tinkerbell workflow: root element must be a map", templateName)
	}

	// Check for required fields
	requiredFields := []string{"name", "version", "tasks"}
	for _, field := range requiredFields {
		if _, exists := resultMap[field]; !exists {
			return inv_errors.Errorf("Template '%s' is missing required field: %s", templateName, field)
		}
	}

	// Validate tasks structure
	tasks, ok := resultMap["tasks"].([]interface{})
	if !ok {
		return inv_errors.Errorf("Template '%s' has invalid 'tasks' field: must be an array", templateName)
	}

	if len(tasks) == 0 {
		return inv_errors.Errorf("Template '%s' has no tasks defined", templateName)
	}

	// Validate each task has required fields
	for i, task := range tasks {
		taskMap, ok := task.(map[string]interface{})
		if !ok {
			return inv_errors.Errorf("Template '%s' task %d is not a valid map", templateName, i)
		}

		taskRequiredFields := []string{"name", "worker", "actions"}
		for _, field := range taskRequiredFields {
			if _, exists := taskMap[field]; !exists {
				return inv_errors.Errorf("Template '%s' task %d is missing required field: %s", templateName, i, field)
			}
		}

		// Validate actions exist
		actions, ok := taskMap["actions"].([]interface{})
		if !ok {
			return inv_errors.Errorf("Template '%s' task %d has invalid 'actions' field: must be an array", templateName, i)
		}

		if len(actions) == 0 {
			return inv_errors.Errorf("Template '%s' task %d has no actions defined", templateName, i)
		}
	}

	return nil
}

// createDummyTemplateData creates dummy data for template rendering validation
func createDummyTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"DeviceInfoHwMacID":                      "00:00:00:00:00:00",
		"DeviceInfoSecurityFeature":              "SECURITY_FEATURE_NONE",
		"DeviceInfoOSImageURL":                   "https://example.com/image.img",
		"DeviceInfoOsImageSHA256":                "placeholder",
		"DeviceInfoOSTLSCACert":                  "placeholder",
		"DeviceInfoUserLVMSize":                  "50",
		"DeviceInfoOSResourceID":                 "os-resource-id",
		"TinkerActionImageSecureBootFlagRead":    "image:latest",
		"TinkerActionImageEraseNonRemovableDisk": "image:latest",
		"TinkerActionImageQemuNbdImage2Disk":     "image:latest",
		"TinkerActionImageWriteFile":             "image:latest",
		"TinkerActionImageCexec":                 "image:latest",
		"TinkerActionImageEfibootset":            "image:latest",
		"TinkerActionImageFdeDmv":                "image:latest",
		"TinkerActionImageKernelUpgrade":         "image:latest",
		"EnvENProxyHTTP":                         "http://proxy:8080",
		"EnvENProxyHTTPS":                        "https://proxy:8443",
		"EnvENProxyNoProxy":                      "localhost",
		"CloudInitData":                          "placeholder",
		"CustomConfigs":                          "",
		"InstallerScript":                        "placeholder",
	}
}

// ValidateAllTemplates validates all templates in the templates map
func ValidateAllTemplates() error {
	for templateName := range templates.TemplatesMap {
		if err := ValidateTemplateYAML(templateName); err != nil {
			return fmt.Errorf("validation failed for template '%s': %w", templateName, err)
		}
	}
	return nil
}

// IsValidTemplateName checks if a template name exists in the templates map
func IsValidTemplateName(templateName string) bool {
	_, exists := templates.TemplatesMap[templateName]
	return exists
}
