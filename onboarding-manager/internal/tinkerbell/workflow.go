// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"context"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
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

// TODO (ITEP-1865): We can probably optimize it.
// Instead of doing GET+CREATE we can try CREATE and check if resource already exists.
//
//nolint:dupl // This is for creating workflow if not exists.
func CreateWorkflowIfNotExists(ctx context.Context, k8sCli client.Client, workflow *tink.Workflow) error {
	got := &tink.Workflow{}
	err := k8sCli.Get(ctx, client.ObjectKeyFromObject(workflow), got)
	if err != nil && errors.IsNotFound(err) {
		zlog.Debug().Msgf("Creating new Tinkerbell workflow %s.", workflow.Name)
		createErr := k8sCli.Create(ctx, workflow)
		if createErr != nil {
			zlog.InfraSec().InfraErr(err).Msgf("")
			return inv_errors.Errorf("Failed to create Tinkerbell workflow %s", workflow.Name)
		}

		return nil
	}

	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("")
		// some other error that may need retry
		return inv_errors.Errorf("Failed to check if Tinkerbell workflow %s exists.", workflow.Name)
	}

	zlog.Debug().Msgf("Tinkerbell workflow %s already exists.", got.Name)

	// err is nil
	return nil
}

func DeleteProdWorkflowResourcesIfExist(ctx context.Context, k8sNamespace, hostUUID string) error {
	zlog.Debug().Msgf("Deleting prod workflow resources for host %s", hostUUID)

	kubeClient, err := K8sClientFactory()
	if err != nil {
		return err
	}

	zlog.Debug().Msgf("Deleting prod template for host %s", hostUUID)

	prodTemplate := &tink.Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetProdTemplateName(hostUUID),
			Namespace: k8sNamespace,
		},
	}

	if err = kubeClient.Delete(ctx, prodTemplate); err != nil && !errors.IsNotFound(err) {
		zlog.InfraSec().InfraErr(err).Msg("")
		zlog.Debug().Msgf("Failed to delete prod template resources for host %s", hostUUID)
		return inv_errors.Errorf("Failed to delete prod template resources for host")
	}

	zlog.Debug().Msgf("Deleting prod workflow for host %s", hostUUID)

	prodWorkflow := &tink.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: "tinkerbell.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetProdWorkflowName(hostUUID),
			Namespace: k8sNamespace,
		},
	}

	if err = kubeClient.Delete(ctx, prodWorkflow); err != nil && !errors.IsNotFound(err) {
		zlog.InfraSec().InfraErr(err).Msg("")
		zlog.Debug().Msgf("Failed to delete prod workflow resources for host %s", hostUUID)
		return inv_errors.Errorf("Failed to delete prod workflow resources for host")
	}

	return nil
}
