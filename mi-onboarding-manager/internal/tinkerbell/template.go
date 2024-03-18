// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"context"
	"fmt"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
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

/*
	see https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/

blob/1a9621b4f8d5146659b680518052a3b7a24d0867/internal/onboardingmgr/onbworkflowclient/workflowcreator.go#L1044.
*/
func GenerateTemplateForProd(k8sNamespace string, deviceInfo utils.DeviceInfo) (*tink.Template, error) {
	tmplName := fmt.Sprintf("%s-%s-prod", deviceInfo.ImType, deviceInfo.GUID)
	var tmplData []byte
	var err error
	switch deviceInfo.ImType {
	case utils.ImgTypeBkc:
		tmplData, err = NewTemplateDataProdBKC(tmplName, deviceInfo.Rootfspart, deviceInfo.RootfspartNo,
			deviceInfo.LoadBalancerIP, deviceInfo.HwIP, deviceInfo.Gateway, deviceInfo.ClientImgName, deviceInfo.ProvisionerIP,
			deviceInfo.SecurityFeature, deviceInfo.ClientID, deviceInfo.ClientSecret, *common.FlagEnableDeviceInitialization,
			deviceInfo.TinkerVersion, deviceInfo.Hostname)
		if err != nil {
			return nil, err
		}
	case utils.ImgTypeFocal:
		tmplData, err = NewTemplateDataProd(tmplName, deviceInfo.Rootfspart,
			deviceInfo.RootfspartNo, deviceInfo.LoadBalancerIP, deviceInfo.ProvisionerIP)
		if err != nil {
			return nil, err
		}
	case utils.ImgTypeFocalMs:
		tmplData, err = NewTemplateDataProdMS(tmplName, deviceInfo.Rootfspart, deviceInfo.RootfspartNo,
			deviceInfo.LoadBalancerIP, deviceInfo.HwIP, deviceInfo.Gateway, deviceInfo.HwMacID, deviceInfo.ProvisionerIP)
		if err != nil {
			return nil, err
		}
	default:
		tmplData, err = NewTemplateDataProd(tmplName, deviceInfo.Rootfspart,
			deviceInfo.RootfspartNo, deviceInfo.LoadBalancerIP, deviceInfo.ProvisionerIP)
		if err != nil {
			return nil, err
		}
	}

	tmpl := NewTemplate(string(tmplData), tmplName, k8sNamespace)
	return tmpl, nil
}

func GenerateTemplateForDI(k8sNamespace string, deviceInfo utils.DeviceInfo) (*tink.Template, error) {
	tmplName := "fdodi-" + deviceInfo.GUID
	tmplData, err := NewTemplateData(tmplName, deviceInfo.HwIP, "CLIENT-SDK-TPM",
		deviceInfo.DiskType, deviceInfo.HwSerialID, deviceInfo.TinkerVersion)
	if err != nil {
		// failed to marshal template data
		zlog.MiSec().MiErr(err).Msg("")
		return nil, inv_errors.Errorf("Failed to generate DI template resources for host %s", deviceInfo.GUID)
	}
	tmpl := NewTemplate(string(tmplData), tmplName, k8sNamespace)

	return tmpl, nil
}

// TODO (LPIO-1865): We can probably optimize it.
// Instead of doing GET+CREATE we can try CREATE and check if resource already exists.
func CreateTemplateIfNotExists(ctx context.Context, k8sCli client.Client, template *tink.Template) error {
	got := &tink.Template{}
	err := k8sCli.Get(ctx, client.ObjectKeyFromObject(template), got)
	if err != nil && errors.IsNotFound(err) {
		zlog.Debug().Msgf("Creating new Tinkerbell template %s.", template.Name)
		createErr := k8sCli.Create(ctx, template)
		if createErr != nil {
			zlog.MiSec().MiErr(err).Msgf("")
			return inv_errors.Errorf("Failed to create Tinkerbell template %s", template.Name)
		}

		return nil
	}

	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("")
		// some other error that may need retry
		return inv_errors.Errorf("Failed to check if Tinkerbell template %s exists.", template.Name)
	}

	zlog.Debug().Msgf("Tinkerbell template %s already exists.", got.Name)

	// already exists, do not return error
	return nil
}
