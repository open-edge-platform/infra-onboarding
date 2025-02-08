/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/invclient"
	om_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/internal/testing"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/onboarding-manager/pkg/status"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/client"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/status"
	inv_testing "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/testing"
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	policyPath := projectRoot + "/build"
	migrationsDir := projectRoot + "/build"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestUpdateHostStatusByHostGuid(t *testing.T) {
	type args struct {
		ctx              context.Context
		tenantID         string
		invClient        *invclient.OnboardingInventoryClient
		hostUUID         string
		statusDetails    string
		onboardingStatus inv_status.ResourceStatus
	}
	om_testing.CreateInventoryOnboardingClientForTesting()
	t.Cleanup(func() {
		om_testing.DeleteInventoryOnboardingClientForTesting()
	})
	host := inv_testing.CreateHost(t, nil, nil)

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Successful Status Update",
			args: args{
				ctx:              context.TODO(),
				tenantID:         client.FakeTenantID,
				invClient:        om_testing.InvClient,
				hostUUID:         host.Uuid,
				statusDetails:    "some detail",
				onboardingStatus: om_status.OnboardingStatusDone,
			},
			wantErr: false,
		},
		{
			name: "InvalidUUIDError",
			args: args{
				ctx:       context.TODO(),
				tenantID:  client.FakeTenantID,
				invClient: om_testing.InvClient,
				hostUUID:  "mockhostUUID",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateHostStatusByHostGUID(tt.args.ctx, tt.args.tenantID, tt.args.invClient, tt.args.hostUUID,
				tt.args.statusDetails, tt.args.onboardingStatus); (err != nil) != tt.wantErr {
				t.Errorf("UpdateHostStatusByHostGUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
