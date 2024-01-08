// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewTemplate(tpData string, name, ns string) *tink.Template {
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
