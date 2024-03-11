// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"context"
	"fmt"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	clientName = "TinkerbellWorkflowHandler"
	zlog       = logging.GetLogger(clientName)
)

func NewWorkflow(name, ns, mac, hardwareRef, templateRef string) *tink.Workflow {
	wf := &tink.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: tink.WorkflowSpec{
			HardwareMap: map[string]string{
				"device_1": mac,
			},
			HardwareRef: hardwareRef,
			TemplateRef: templateRef,
		},
	}

	return wf
}

// TODO (LPIO-1865): We can probably optimize it. Instead of doing GET+CREATE we can try CREATE and check if resource already exists.
func CreateWorkflowIfNotExists(ctx context.Context, k8sCli client.Client, workflow *tink.Workflow) error {
	got := &tink.Workflow{}
	err := k8sCli.Get(ctx, client.ObjectKeyFromObject(workflow), got)
	if err != nil && errors.IsNotFound(err) {
		zlog.Debug().Msgf("Creating new Tinkerbell workflow %s.", workflow.Name)
		return k8sCli.Create(ctx, workflow)
	}

	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("")
		// some other error that may need retry
		return inv_errors.Errorf("Failed to check if Tinkerbell workflow %s exists.", workflow.Name)
	}

	zlog.Debug().Msgf("Tinkerbell workflow %s already exists.", got.Name)

	// err is nil
	return nil
}

func DeleteProdWorkflowResourcesIfExist(ctx context.Context, k8sNamespace string, hostUUID string) error {
	zlog.Info().Msgf("Deleting prod workflow resources for host %s", hostUUID)

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	zlog.Info().Msgf("Deleting prod template for host %s", hostUUID)

	diTemplate := &tink.Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-prod", hostUUID),
			Namespace: k8sNamespace,
		},
	}

	if err = kubeClient.Delete(ctx, diTemplate); err != nil && !errors.IsNotFound(err) {
		zlog.MiSec().MiErr(err).Msg("")
		return inv_errors.Errorf("Failed to delete prod template resources for host %s", hostUUID)
	}

	zlog.Info().Msgf("Deleting prod workflow for host %s", hostUUID)

	diWorkflow := &tink.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("workflow-%s-prod", hostUUID),
			Namespace: k8sNamespace,
		},
	}

	if err = kubeClient.Delete(ctx, diWorkflow); err != nil && !errors.IsNotFound(err) {
		zlog.MiSec().MiErr(err).Msg("")
		return inv_errors.Errorf("Failed to delete prod workflow resources for host %s", hostUUID)
	}

	return nil
}

func DeleteDIWorkflowResourcesIfExist(ctx context.Context, k8sNamespace string, hostUUID string) error {
	zlog.Info().Msgf("Deleting DI template for host %s", hostUUID)

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	diTemplate := &tink.Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fdodi-" + hostUUID,
			Namespace: k8sNamespace,
		},
	}

	if err = kubeClient.Delete(ctx, diTemplate); err != nil && !errors.IsNotFound(err) {
		zlog.MiSec().MiErr(err).Msg("")
		return inv_errors.Errorf("Failed to delete DI template resources for host %s", hostUUID)
	}

	zlog.Info().Msgf("Deleting DI workflow for host %s", hostUUID)

	diWorkflow := &tink.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workflow-" + hostUUID,
			Namespace: k8sNamespace,
		},
	}

	if err = kubeClient.Delete(ctx, diWorkflow); err != nil && !errors.IsNotFound(err) {
		zlog.MiSec().MiErr(err).Msg("")
		return inv_errors.Errorf("Failed to delete DI workflow resources for host %s", hostUUID)
	}

	return nil
}
