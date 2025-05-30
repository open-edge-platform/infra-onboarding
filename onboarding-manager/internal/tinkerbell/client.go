// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"context"
	"time"

	tinkv1alpha1 "github.com/tinkerbell/tink/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
)

const defaultK8sClientTimeout = 3 * time.Second

var K8sClientFactory = newK8SClient

func newK8SClient() (client.Client, error) {
	logf.SetLogger(zap.New(zap.WriteTo(zlog)))

	config, err := rest.InClusterConfig()
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, inv_errors.Errorf("Cannot create K8s config for client")
	}

	if schemeErr := tinkv1alpha1.AddToScheme(scheme.Scheme); schemeErr != nil {
		zlog.InfraSec().InfraErr(schemeErr).Msg("")
		return nil, inv_errors.Errorf("Cannot add Tink schema for K8s client")
	}

	kubeClient, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, inv_errors.Errorf("Unable to create new K8s client")
	}
	return kubeClient, nil
}

func ListTemplates() ([]tinkv1alpha1.Template, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sClientTimeout)
	defer cancel()

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return nil, err
	}

	tmplList := &tinkv1alpha1.TemplateList{}
	err = kubeClient.List(ctx, tmplList, client.InNamespace(env.K8sNamespace))
	if err != nil {
		return nil, err
	}

	return tmplList.Items, nil
}

func CreateTemplate(template *tinkv1alpha1.Template) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sClientTimeout)
	defer cancel()

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	zlog.Info().Msgf("Creating new Tinkerbell template %s.", template.Name)

	createErr := kubeClient.Create(ctx, template)
	if createErr != nil {
		zlog.InfraSec().InfraErr(createErr).Msgf("")
		return inv_errors.Errorf("Failed to create Tinkerbell template %s", template.Name)
	}

	zlog.Debug().Msgf("Tinkerbell template %q created", template.Name)

	return nil
}

func DeleteTemplate(name, namespace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sClientTimeout)
	defer cancel()

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	zlog.Debug().Msgf("Deleting prod template %s in namespace %s", name, namespace)

	template := &tinkv1alpha1.Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err = kubeClient.Delete(ctx, template); err != nil && !errors.IsNotFound(err) {
		zlog.InfraSec().InfraErr(err).Msg("")
		return inv_errors.Errorf("Failed to delete Tinkerbell template")
	}

	return nil
}
