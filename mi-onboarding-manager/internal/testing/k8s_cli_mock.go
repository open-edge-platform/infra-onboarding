// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package testing

import (
	"context"
	"strings"

	inv_client "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/client"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
)

type k8sCliMock struct {
	mock.Mock

	// withInventory use real Inventory client to get current OS to fill in the Tink Hardware CRD object
	withInventory bool
}

func (k *k8sCliMock) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := k.Called()

	if strings.HasPrefix(key.Name, "workflow-") {
		workflow := obj.(*tink.Workflow)
		workflow.Status.State = tink.WorkflowStateSuccess
	}

	if strings.HasPrefix(key.Name, "machine-") {
		// If workfkow state is SUCCESS (see above), OM fetches Tink hardware to update current OS of Instance.
		// For testing purpose, and to avoid NotFound OS resource during update, we set current OS to Instance's desired OS.
		// Therefore, we retrieve existing Instance to get desired OS and set OsSlug to its resource ID.
		// This behavior can be controlled by withInventory flag.

		osResourceID := "os-12345678"

		if k.withInventory {
			hostUUID := strings.TrimPrefix(key.Name, "machine-")
			host, _ := InvClient.Client.GetHostByUUID(context.Background(), inv_client.FakeTenantID, hostUUID)
			osResourceID = host.GetInstance().GetDesiredOs().GetResourceId()
		}

		hardware := obj.(*tink.Hardware)
		hardware.Spec.Metadata = &tink.HardwareMetadata{
			Instance: &tink.MetadataInstance{
				OperatingSystem: &tink.MetadataInstanceOperatingSystem{
					OsSlug: osResourceID,
				},
			},
		}
	}

	return args.Error(0)
}

func (k *k8sCliMock) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *k8sCliMock) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *k8sCliMock) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *k8sCliMock) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *k8sCliMock) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *k8sCliMock) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *k8sCliMock) Status() client.SubResourceWriter {
	args := k.Called()
	return args.Get(0).(client.SubResourceWriter)
}

func (k *k8sCliMock) SubResource(subResource string) client.SubResourceClient {
	args := k.Called()
	return args.Get(0).(client.SubResourceClient)
}

func (k *k8sCliMock) Scheme() *runtime.Scheme {
	args := k.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (k *k8sCliMock) RESTMapper() meta.RESTMapper {
	args := k.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (k *k8sCliMock) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := k.Called()
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (k *k8sCliMock) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	args := k.Called()
	return args.Get(0).(bool), args.Error(1)
}

func K8sCliMockFactory(createShouldFail, getShouldFail, deleteShouldFail, useRealInventory bool) func() (client.Client, error) {
	k8sMock := &k8sCliMock{
		withInventory: useRealInventory,
	}

	if createShouldFail {
		k8sMock.On("Create", mock.Anything, mock.Anything, mock.Anything).Return("", "", errors.New(""))
	} else {
		k8sMock.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}

	if getShouldFail {
		k8sMock.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(k8s_errors.NewBadRequest(""))
	} else {
		k8sMock.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}

	if deleteShouldFail {
		k8sMock.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(inv_errors.Errorf(""))
	} else {
		k8sMock.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}

	return func() (client.Client, error) {
		return k8sMock, nil
	}
}
