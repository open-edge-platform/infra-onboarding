/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package onboarding

import (
	"context"

	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/client/cache"
)

type MockInventoryClient struct {
	mock.Mock
}

func (m *MockInventoryClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockInventoryClient) List(ctx context.Context, filter *inv_v1.ResourceFilter) (*inv_v1.ListResourcesResponse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*inv_v1.ListResourcesResponse), args.Error(1)
}

func (m *MockInventoryClient) ListAll(ctx context.Context, filter *inv_v1.ResourceFilter) ([]*inv_v1.Resource, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*inv_v1.Resource), args.Error(1)
}

func (m *MockInventoryClient) Find(ctx context.Context, filter *inv_v1.ResourceFilter) (*inv_v1.FindResourcesResponse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*inv_v1.FindResourcesResponse), args.Error(1)
}

func (m *MockInventoryClient) FindAll(ctx context.Context, filter *inv_v1.ResourceFilter) ([]string, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockInventoryClient) Get(ctx context.Context, id string) (*inv_v1.GetResourceResponse, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*inv_v1.GetResourceResponse), args.Error(1)
}

func (m *MockInventoryClient) Create(ctx context.Context, resource *inv_v1.Resource) (*inv_v1.CreateResourceResponse, error) {
	args := m.Called(ctx, resource)
	return args.Get(0).(*inv_v1.CreateResourceResponse), args.Error(1)
}

func (m *MockInventoryClient) Update(ctx context.Context, id string,
	mask *fieldmaskpb.FieldMask, resource *inv_v1.Resource,
) (*inv_v1.UpdateResourceResponse, error) {
	args := m.Called(ctx, id, mask, resource)
	return args.Get(0).(*inv_v1.UpdateResourceResponse), args.Error(1)
}

func (m *MockInventoryClient) Delete(ctx context.Context, id string) (*inv_v1.DeleteResourceResponse, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*inv_v1.DeleteResourceResponse), args.Error(1)
}

func (m *MockInventoryClient) UpdateSubscriptions(ctx context.Context, kinds []inv_v1.ResourceKind) error {
	args := m.Called(ctx, kinds)
	return args.Error(0)
}

func (m *MockInventoryClient) ListInheritedTelemetryProfiles(ctx context.Context,
	inheritBy *inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy,
	filter string,
	orderBy string,
	limit, offset uint32,
) (*inv_v1.ListInheritedTelemetryProfilesResponse, error) {
	args := m.Called(ctx, inheritBy, filter, orderBy, limit, offset)
	return args.Get(0).(*inv_v1.ListInheritedTelemetryProfilesResponse), args.Error(1)
}

func (m *MockInventoryClient) GetHostByUUID(ctx context.Context, uuid string) (*computev1.HostResource, error) {
	args := m.Called(ctx, uuid)
	return args.Get(0).(*computev1.HostResource), args.Error(1)
}

func (m *MockInventoryClient) TestingOnlySetClient(client inv_v1.InventoryServiceClient) {
	m.Called(client)
}

func (m *MockInventoryClient) TestGetClientCache() *cache.InventoryCache {
	m.Called()
	return nil
}

func (m *MockInventoryClient) TestGetClientCacheUUID() *cache.InventoryCache {
	m.Called()
	return nil
}

type MockInvClient struct {
	mock.Mock
}

func (m *MockInvClient) UpdateHostStatusByHostGUID(ctx context.Context, hostGUID string, status computev1.HostStatus) error {
	args := m.Called(ctx, hostGUID, status)
	return args.Error(0)
}

type MockDeviceInfo struct {
	mock.Mock
}

func (m *MockDeviceInfo) HwIP() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDeviceInfo) HwSerialID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDeviceInfo) ProvisionerIP() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDeviceInfo) GUID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDeviceInfo) SetGUID(guid string) {
	m.Called(guid)
}

// type MockMakeGETRequestWithRetry struct {
// 	mock.Mock
// }

// func (m *MockMakeGETRequestWithRetry) Call(deviceInfo *utils.DeviceInfo, caCertPath, certPath string) error {
// 	args := m.Called(deviceInfo.HwSerialID, deviceInfo.ProvisionerIp, caCertPath, certPath, deviceInfo.Guid)
// 	return args.Error(0)
// }

type MockHTTPUtils struct {
	mock.Mock
}

func (m *MockHTTPUtils) MakeHTTPGETRequest(pdip, guid, caCertPath, certPath string) ([]byte, error) {
	args := m.Called(pdip, guid, caCertPath, certPath)
	return args.Get(0).([]byte), args.Error(1)
}

type MockController struct {
	mock.Mock
}

func (m *MockController) Stop() {
	m.Called()
}
