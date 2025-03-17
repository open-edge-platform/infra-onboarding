// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	inv_client "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

type MockK8sClient struct {
	mock.Mock

	// withInventory use real Inventory client to get current OS to fill in the Tink Hardware CRD object
	withInventory bool
}

func (k *MockK8sClient) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	args := k.Called()

	if strings.HasPrefix(key.Name, "workflow-") {
		workflow, ok := obj.(*tink.Workflow)
		if !ok {
			return args.Error(0)
		}
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
			host, err := InvClient.Client.GetHostByUUID(context.Background(), inv_client.FakeTenantID, hostUUID)
			if err != nil {
				fmt.Println(err)
			}
			osResourceID = host.GetInstance().GetDesiredOs().GetResourceId()
		}

		hardware, ok := obj.(*tink.Hardware)
		if !ok {
			return args.Error(0)
		}
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

func (k *MockK8sClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *MockK8sClient) Create(_ context.Context, _ client.Object, _ ...client.CreateOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *MockK8sClient) Delete(_ context.Context, _ client.Object, _ ...client.DeleteOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *MockK8sClient) Update(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *MockK8sClient) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *MockK8sClient) DeleteAllOf(_ context.Context, _ client.Object, _ ...client.DeleteAllOfOption) error {
	args := k.Called()
	return args.Error(0)
}

func (k *MockK8sClient) Status() client.SubResourceWriter {
	args := k.Called()
	return args.Get(0).(client.SubResourceWriter)
}

func (k *MockK8sClient) SubResource(_ string) client.SubResourceClient {
	args := k.Called()
	return args.Get(0).(client.SubResourceClient)
}

func (k *MockK8sClient) Scheme() *runtime.Scheme {
	args := k.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (k *MockK8sClient) RESTMapper() meta.RESTMapper {
	args := k.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (k *MockK8sClient) GroupVersionKindFor(_ runtime.Object) (schema.GroupVersionKind, error) {
	args := k.Called()
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (k *MockK8sClient) IsObjectNamespaced(_ runtime.Object) (bool, error) {
	args := k.Called()
	return args.Get(0).(bool), args.Error(1)
}

func K8sCliMockFactory(createShouldFail, getShouldFail, deleteShouldFail, useRealInventory bool) func() (client.Client, error) {
	k8sMock := &MockK8sClient{
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
