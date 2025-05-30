// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

type MockK8sClient struct {
	mock.Mock
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
	result, ok := args.Get(0).(client.SubResourceWriter)
	if !ok {
		return nil
	}
	return result
}

func (k *MockK8sClient) SubResource(_ string) client.SubResourceClient {
	args := k.Called()
	result, ok := args.Get(0).(client.SubResourceClient)
	if !ok {
		return nil
	}
	return result
}

func (k *MockK8sClient) Scheme() *runtime.Scheme {
	args := k.Called()
	result, ok := args.Get(0).(*runtime.Scheme)
	if !ok {
		return nil
	}
	return result
}

func (k *MockK8sClient) RESTMapper() meta.RESTMapper {
	args := k.Called()
	result, ok := args.Get(0).(meta.RESTMapper)
	if !ok {
		return nil
	}
	return result
}

func (k *MockK8sClient) GroupVersionKindFor(_ runtime.Object) (schema.GroupVersionKind, error) {
	args := k.Called()
	result, ok := args.Get(0).(schema.GroupVersionKind)
	if !ok {
		return schema.GroupVersionKind{}, inv_errors.Errorf("unexpected type for GroupVersionKind: %T", args.Get(0))
	}
	return result, args.Error(1)
}

func (k *MockK8sClient) IsObjectNamespaced(_ runtime.Object) (bool, error) {
	args := k.Called()
	result, ok := args.Get(0).(bool)
	if !ok {
		return false, inv_errors.Errorf("unexpected type for bool: %T", args.Get(0))
	}
	return result, args.Error(1)
}

func K8sCliMockFactory(createShouldFail, getShouldFail, deleteShouldFail bool) func() (client.Client, error) {
	k8sMock := &MockK8sClient{}

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

	k8sMock.On("List", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return func() (client.Client, error) {
		return k8sMock, nil
	}
}
