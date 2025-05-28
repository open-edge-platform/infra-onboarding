// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"context"
	tinkv1alpha1 "github.com/tinkerbell/tink/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

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

func CreateTemplate(template *tinkv1alpha1.Template) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	zlog.Info().Msgf("Creating new Tinkerbell template %q", template.Name)

	createErr := kubeClient.Create(ctx, template)
	if createErr != nil {
		zlog.InfraSec().InfraErr(createErr).Msgf("")
		return inv_errors.Errorf("Failed to create Tinkerbell template %s", template.Name)
	}

	zlog.Debug().Msgf("Tinkerbell template %q created", template.Name)

	return nil
}

func DeleteAllTemplates(namespace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	zlog.Debug().Msgf("Deleting all Tinkerbell templates in namespace %q", namespace)

	if err = kubeClient.DeleteAllOf(ctx, &tinkv1alpha1.Template{}, client.InNamespace(namespace)); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return inv_errors.Errorf("Failed to delete Tinkerbell templates")
	}

	return nil
}
