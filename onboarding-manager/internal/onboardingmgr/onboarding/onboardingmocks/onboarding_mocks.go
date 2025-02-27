/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package onboarding

import (
	"context"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pb "github.com/intel/infra-onboarding/onboarding-manager/pkg/api"
)

type MockNonInteractiveOnboardingServiceOnboardNodeStreamServer struct {
	mock.Mock
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) Send(response *pb.OnboardStreamResponse) error {
	args := m.Called(response)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) Recv() (*pb.OnboardStreamRequest, error) {
	args := m.Called()
	return args.Get(0).(*pb.OnboardStreamRequest), args.Error(1)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) SetHeader(md metadata.MD) error {
	args := m.Called(md)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) SendHeader(md metadata.MD) error {
	args := m.Called(md)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) SetTrailer(md metadata.MD) {
	m.Called(md)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) Context() context.Context {
	args := m.Called()
	return args.Get(0).(context.Context)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) SendMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockNonInteractiveOnboardingServiceOnboardNodeStreamServer) RecvMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

type MockClient struct {
	mock.Mock
}

//

func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := m.Called(obj)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (m *MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	args := m.Called(obj)
	return args.Bool(0), args.Error(1)
}

func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Status() client.SubResourceWriter {
	args := m.Called()
	return args.Get(0).(client.SubResourceWriter)
}

func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	args := m.Called(subResource)
	return args.Get(0).(client.SubResourceClient)
}
