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
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
)

const defaultK8sClientTimeout = 3 * time.Second

var (
	K8sClientFactory = newK8SClient

	clientName = "TinkerbellWorkflowHandler"
	zlog       = logging.GetLogger(clientName)
)

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

func NewWorkflow(name, ns, hardwareRef, templateRef string, hardwareMap map[string]string) *tinkv1alpha1.Workflow {
	wf := &tinkv1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: tinkv1alpha1.WorkflowSpec{
			HardwareMap: hardwareMap,
			HardwareRef: hardwareRef,
			TemplateRef: templateRef,
		},
	}

	return wf
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

func CreateTemplate(k8sNamespace, name string, rawTemplateData []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sClientTimeout)
	defer cancel()

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	zlog.Info().Msgf("Creating new Tinkerbell template %s.", name)

	tpData := string(rawTemplateData)
	template := &tinkv1alpha1.Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k8sNamespace,
		},
		Spec: tinkv1alpha1.TemplateSpec{
			Data: &tpData,
		},
	}

	createErr := kubeClient.Create(ctx, template)
	if createErr != nil {
		zlog.InfraSec().InfraErr(createErr).Msgf("")
		return inv_errors.Errorf("Failed to create Tinkerbell template %s", template.Name)
	}

	zlog.Debug().Msgf("Tinkerbell template %q created", template.Name)

	return nil
}

func CreateHardwareIfNotExists(k8sNamespace, hwName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sClientTimeout)
	defer cancel()

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	hwInfo := &tinkv1alpha1.Hardware{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Hardware",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hwName,
			Namespace: k8sNamespace,
		},
		Spec: tinkv1alpha1.HardwareSpec{},
	}

	createErr := kubeClient.Create(ctx, hwInfo)
	if createErr != nil && !errors.IsAlreadyExists(createErr) {
		zlog.InfraSec().InfraErr(createErr).Msgf("")
		return inv_errors.Errorf("Failed to create Tinkerbell hardware %s", hwInfo.Name)
	}

	return nil
}

func DeleteHardware(k8sNamespace, hwName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultK8sClientTimeout)
	defer cancel()

	zlog.Debug().Msgf("Deleting Tinkerbell Hardware %q", hwName)

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	hw := &tinkv1alpha1.Hardware{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Hardware",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hwName,
			Namespace: k8sNamespace,
		},
	}

	if err = kubeClient.Delete(ctx, hw); err != nil && !errors.IsNotFound(err) {
		zlog.InfraSec().InfraErr(err).Msg("")
		zlog.Debug().Msgf("Failed to delete Tink hardware resource %q", hwName)
		return inv_errors.Errorf("Failed to delete Tink hardware resource")
	}

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

func CreateWorkflowIfNotExists(ctx context.Context, k8sCli client.Client, workflow *tinkv1alpha1.Workflow) error {
	zlog.Info().Msgf("Creating new Tinkerbell workflow %s.", workflow.Name)
	createErr := k8sCli.Create(ctx, workflow)
	if createErr != nil && !errors.IsAlreadyExists(createErr) {
		zlog.InfraSec().InfraErr(createErr).Msgf("")
		return inv_errors.Errorf("Failed to create Tinkerbell workflow %s", workflow.Name)
	}

	zlog.Debug().Msgf("Tinkerbell workflow %q created successfully.", workflow.Name)

	return nil
}

func DeleteWorkflowIfExists(ctx context.Context, k8sNamespace, workflowName string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultK8sClientTimeout)
	defer cancel()

	zlog.Info().Msgf("Deleting Tinkerbell Workflow %q", workflowName)

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	prodWorkflow := &tinkv1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflowName,
			Namespace: k8sNamespace,
		},
	}

	if err = kubeClient.Delete(ctx, prodWorkflow); err != nil && !errors.IsNotFound(err) {
		zlog.InfraSec().InfraErr(err).Msg("")
		zlog.Debug().Msgf("Failed to delete Tinkerbell Workflow %q", workflowName)
		return inv_errors.Errorf("Failed to delete Tinkerbell Workflow")
	}

	return nil
}
