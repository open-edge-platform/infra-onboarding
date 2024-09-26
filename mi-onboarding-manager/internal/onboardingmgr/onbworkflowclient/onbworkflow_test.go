/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package onbworkflowclient_test

import (
	"context"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/onbworkflowclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/tinkerbell"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/flags"
)

func TestCheckStatusOrRunProdWorkflow(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	currFlagEnableDeviceInitialization := *flags.FlagDisableCredentialsManagement
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
		*common.FlagEnableDeviceInitialization = currFlagEnableDeviceInitialization
	}()
	*common.FlagEnableDeviceInitialization = true
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, true, false)
	type args struct {
		ctx        context.Context
		deviceInfo utils.DeviceInfo
		instance   *computev1.InstanceResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "CheckStatusOrRunDIWorkflow_WithValidCertificates",
			args: args{
				ctx: context.Background(),
				instance: &computev1.InstanceResource{
					Host: &computev1.HostResource{
						ResourceId: "host-084d9b08",
					},
					DesiredOs: &osv1.OperatingSystemResource{},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := onbworkflowclient.CheckStatusOrRunProdWorkflow(tt.args.ctx, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckStatusOrRunProdWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckStatusOrRunRebootWorkflow(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	currFlagEnableDeviceInitialization := *flags.FlagDisableCredentialsManagement
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
		*common.FlagEnableDeviceInitialization = currFlagEnableDeviceInitialization
	}()
	*common.FlagEnableDeviceInitialization = false
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, false, false)
	type args struct {
		ctx        context.Context
		deviceInfo utils.DeviceInfo
		instance   *computev1.InstanceResource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "HostStatusNotReady",
			args: args{
				ctx:      context.Background(),
				instance: &computev1.InstanceResource{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := onbworkflowclient.CheckStatusOrRunRebootWorkflow(tt.args.ctx, tt.args.deviceInfo, tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("CheckStatusOrRunRebootWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
