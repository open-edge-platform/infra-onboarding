// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"context"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell/templates"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
)

var (
	OSProfileToTemplateName = map[string]string{
		"microvisor-nonrt": templates.MicrovisorName,
		"microvisor-rt":    templates.MicrovisorName,
	}
)

func NewTemplate(tpData, name, ns string) *tink.Template {
	tp := &tink.Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: tink.TemplateSpec{
			Data: &tpData,
		},
	}
	return tp
}

func GenerateTemplateForProd(
	k8sNamespace string,
	deviceInfo onboarding_types.DeviceInfo,
) (*tink.Template, error) {
	templateName, exists := OSProfileToTemplateName[deviceInfo.OsProfileName]
	if !exists {
		return nil, inv_errors.Errorfc(codes.InvalidArgument,
			"Unsupported OS profile %s", deviceInfo.OsProfileName)
	}

	templateSpec, exists := templates.TemplatesMap[templateName]
	if !exists {
		// this should never happen
		return nil, inv_errors.Errorfc(codes.Internal,
			"Unsupported template name %s", templateName)
	}

	tmpl := NewTemplate(string(templateSpec), templateName, k8sNamespace)
	return tmpl, nil
}

// TODO (ITEP-1865): We can probably optimize it.
//
//	Instead of doing GET+CREATE we can try CREATE and check if resource already exists.
//
//nolint:dupl // This is for creating template if not exists.
func CreateTemplateIfNotExists(ctx context.Context, k8sCli client.Client, template *tink.Template) error {
	got := &tink.Template{}
	err := k8sCli.Get(ctx, client.ObjectKeyFromObject(template), got)
	if err != nil && errors.IsNotFound(err) {
		zlog.Debug().Msgf("Creating new Tinkerbell template %s.", template.Name)
		createErr := k8sCli.Create(ctx, template)
		if createErr != nil {
			zlog.InfraSec().InfraErr(createErr).Msgf("")
			return inv_errors.Errorf("Failed to create Tinkerbell template %s", template.Name)
		}

		return nil
	}

	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("")
		// some other error that may need retry
		return inv_errors.Errorf("Failed to check if Tinkerbell template %s exists.", template.Name)
	}

	zlog.Debug().Msgf("Tinkerbell template %s already exists.", got.Name)

	// already exists, do not return error
	return nil
}
