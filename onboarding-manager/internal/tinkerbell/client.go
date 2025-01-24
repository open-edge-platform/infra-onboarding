// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	tinkv1alpha1 "github.com/tinkerbell/tink/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/errors"
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
