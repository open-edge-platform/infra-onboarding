// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell_test

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell/templates"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"gopkg.in/yaml.v2"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	dkam_testing "github.com/open-edge-platform/infra-onboarding/dkam/testing"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell"
)

func TestWorkflowActionToStatusDetail(t *testing.T) {
	ctx := context.Background() // Define context
	dkam_testing.PrepareTestCaCertificateFile(t)
	dkam_testing.PrepareTestInfraConfig(t)

	tmpl := tinkerbell.NewTemplate(string(templates.UbuntuTemplate), "test-template", "test-namespace")

	wfInputs, err := tinkerbell.GenerateWorkflowInputs(ctx, onboarding_types.DeviceInfo{
		OsType:           osv1.OsType_OS_TYPE_MUTABLE,
		TenantID:         "test-tenantid",
		Hostname:         "test-hostname",
		AuthClientID:     "test-client-id",
		AuthClientSecret: "test-client-secret",
		HwMacID:          "aa:bb:cc:dd:ee:ff",
		PlatformBundle:   "null",
	})
	require.NoError(t, err)

	wf := tinkerbell.NewWorkflow("test-wf", "test-namespace",
		"test-hardware", tmpl.Name, wfInputs)

	rawWf, err := yaml.Marshal(wf)

	prodBkcWorkflowInstance, err := unmarshalWorkflow(rawWf)
	require.NoError(t, err)

	workflows := []*tinkerbell.Workflow{
		prodBkcWorkflowInstance,
	}

	for _, wf := range workflows {
		t.Run(wf.Name, func(t *testing.T) {
			for _, action := range wf.Tasks[0].Actions {
				_, exists := tinkerbell.WorkflowStepToStatusDetail[action.Name]
				assert.True(t, exists)
				if !exists {
					t.Errorf("No status detail for action %q", action.Name)
				}
			}
		})
	}
}

func unmarshalWorkflow(yamlContent []byte) (*tinkerbell.Workflow, error) {
	var workflow tinkerbell.Workflow

	if err := yaml.Unmarshal(yamlContent, &workflow); err != nil {
		return &tinkerbell.Workflow{}, errors.Wrap(err, "parsing yaml data")
	}
	return &workflow, nil
}

func TestGenerateStatusDetailFromWorkflowState(t *testing.T) {
	tests := getStaticWorkflowTests()           // Static test cases
	tests = append(tests, getDynamicTests()...) // Dynamic test cases

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tinkerbell.GenerateStatusDetailFromWorkflowState(tt.args.workflow),
				"GenerateStatusDetailFromWorkflowState(%v)", tt.args.workflow)
		})
	}
}

// getStaticWorkflowTests returns predefined test cases.
func getStaticWorkflowTests() []struct {
	name string
	args struct{ workflow *tink.Workflow }
	want string
} {
	return []struct {
		name string
		args struct{ workflow *tink.Workflow }
		want string
	}{
		{"Workflow nil", struct{ workflow *tink.Workflow }{nil}, ""},
		{"No workflow tasks", struct{ workflow *tink.Workflow }{&tink.Workflow{}}, ""},
		{
			"No workflow actions",
			struct{ workflow *tink.Workflow }{
				&tink.Workflow{Status: tink.WorkflowStatus{Tasks: []tink.Task{{}}}},
			},
			"",
		},
		{
			"Empty action",
			struct{ workflow *tink.Workflow }{
				&tink.Workflow{Status: tink.WorkflowStatus{Tasks: []tink.Task{{Actions: []tink.Action{}}}}},
			},
			"",
		},
		{
			"Successful workflow",
			struct{ workflow *tink.Workflow }{
				&tink.Workflow{Status: tink.WorkflowStatus{
					State: tink.WorkflowStateSuccess, Tasks: []tink.Task{{Actions: []tink.Action{{}}}},
				}},
			},
			"",
		},
		{
			"SingleAction_Success",
			struct{ workflow *tink.Workflow }{
				&tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{{Actions: []tink.Action{{
						Name:   tinkerbell.ActionReboot,
						Status: tink.WorkflowStateSuccess,
					}}}},
				}},
			},
			fmt.Sprintf("1/1: %s", tinkerbell.WorkflowStepToStatusDetail[tinkerbell.ActionReboot]),
		},
		{
			"Multiple actions - workflow not completed",
			struct{ workflow *tink.Workflow }{
				&tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{{
						Actions: []tink.Action{
							{Name: tinkerbell.ActionFdeEncryption, Status: tink.WorkflowStateSuccess},
							{Name: tinkerbell.ActionAddAptProxy, Status: tink.WorkflowStateSuccess},
							{Name: tinkerbell.ActionReboot, Status: tink.WorkflowStateRunning},
						},
					}},
				}},
			},
			fmt.Sprintf("3/3: %s", tinkerbell.WorkflowStepToStatusDetail[tinkerbell.ActionReboot]),
		},
		{
			"Failed action with message",
			struct{ workflow *tink.Workflow }{
				&tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{{Actions: []tink.Action{
						{Name: tinkerbell.ActionFdeEncryption, Status: tink.WorkflowStateSuccess},
						{Name: tinkerbell.ActionAddAptProxy, Status: tink.WorkflowStateFailed, Message: "some message"},
					}}},
				}},
			},
			fmt.Sprintf("2/2: %s failed: some message", tinkerbell.WorkflowStepToStatusDetail[tinkerbell.ActionAddAptProxy]),
		},
		{
			"Failed action without message",
			struct{ workflow *tink.Workflow }{
				&tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{{Actions: []tink.Action{
						{Name: tinkerbell.ActionFdeEncryption, Status: tink.WorkflowStateSuccess},
						{Name: tinkerbell.ActionAddAptProxy, Status: tink.WorkflowStateFailed},
					}}},
				}},
			},
			fmt.Sprintf("2/2: %s failed", tinkerbell.WorkflowStepToStatusDetail[tinkerbell.ActionAddAptProxy]),
		},
		{
			"Timed out action",
			struct{ workflow *tink.Workflow }{
				&tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{{Actions: []tink.Action{
						{Name: tinkerbell.ActionFdeEncryption, Status: tink.WorkflowStateSuccess},
						{Name: tinkerbell.ActionAddAptProxy, Status: tink.WorkflowStateTimeout, Message: "some message"},
					}}},
				}},
			},
			fmt.Sprintf("2/2: %s timeout", tinkerbell.WorkflowStepToStatusDetail[tinkerbell.ActionAddAptProxy]),
		},
	}
}

func getDynamicTests() []struct {
	name string
	args struct {
		workflow *tink.Workflow
	}
	want string
} {
	tests := make([]struct {
		name string
		args struct {
			workflow *tink.Workflow
		}
		want string
	}, 3)

	for action, detail := range tinkerbell.WorkflowStepToStatusDetail {
		tests = append(tests, struct {
			name string
			args struct {
				workflow *tink.Workflow
			}
			want string
		}{
			name: fmt.Sprintf("SingleAction_%s_Success", action),
			args: struct{ workflow *tink.Workflow }{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{Name: action, Status: tink.WorkflowStateSuccess},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("1/1: %s", detail),
		})
	}
	return tests
}
