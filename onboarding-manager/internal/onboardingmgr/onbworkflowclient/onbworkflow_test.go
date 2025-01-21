/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package onbworkflowclient_test

import (
	"context"
	"testing"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/compute/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/onboardingmgr/onbworkflowclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/onboardingmgr/utils"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/testing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/tinkerbell"
)

func TestCheckStatusOrRunProdWorkflow(t *testing.T) {
	currK8sClientFactory := tinkerbell.K8sClientFactory
	defer func() {
		tinkerbell.K8sClientFactory = currK8sClientFactory
	}()
	tinkerbell.K8sClientFactory = om_testing.K8sCliMockFactory(false, true, false, false)
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
