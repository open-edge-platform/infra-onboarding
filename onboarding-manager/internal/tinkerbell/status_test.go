// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"fmt"
	"testing"

	"github.com/intel/infra-onboarding/onboarding-manager/internal/onboardingmgr/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tink "github.com/tinkerbell/tink/api/v1alpha1"
)

func TestWorkflowActionToStatusDetail(t *testing.T) {
	prodBkcWorkflow, err := NewTemplateDataUbuntu("test-prod-bkc", utils.DeviceInfo{})
	require.NoError(t, err)

	prodBkcWorkflowInstance, err := unmarshalWorkflow(prodBkcWorkflow)
	require.NoError(t, err)

	workflows := []*Workflow{
		prodBkcWorkflowInstance,
	}

	for _, wf := range workflows {
		t.Run(wf.Name, func(t *testing.T) {
			for _, action := range wf.Tasks[0].Actions {
				_, exists := WorkflowStepToStatusDetail[action.Name]
				assert.True(t, exists)
				if !exists {
					t.Errorf("No status detail for action %q", action.Name)
				}
			}
		})
	}
}

func TestGenerateStatusDetailFromWorkflowState(t *testing.T) {
	type args struct {
		workflow *tink.Workflow
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Workflow nil",
			args: args{
				workflow: nil,
			},
			want: "",
		},
		{
			name: "No workflow tasks",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{}},
			},
			want: "",
		},
		{
			name: "No workflow actions",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{},
							},
						},
					},
				}},
			},
			want: "",
		},
		{
			name: "Empty action",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{},
						},
					},
				}},
			},
			want: "",
		},
		{
			name: "Successful workflow",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					State: tink.WorkflowStateSuccess,
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{},
							},
						},
					},
				}},
			},
			want: "",
		},
		{
			name: "SingleAction_Success",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionReboot,
									Status: tink.WorkflowStateSuccess,
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("1/1: %s", WorkflowStepToStatusDetail[ActionReboot]),
		},
		{
			name: "Multiple actions - workflow not completed 0",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionAddAptProxy,
									Status: "",
								},
								{
									Name:   ActionReboot,
									Status: "",
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("1/2: %s", WorkflowStepToStatusDetail[ActionAddAptProxy]),
		},
		{
			name: "Multiple actions - workflow not completed 1",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionFdeEncryption,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:   ActionAddAptProxy,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:   ActionReboot,
									Status: "",
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("3/3: %s", WorkflowStepToStatusDetail[ActionReboot]),
		},
		{
			name: "Multiple actions - workflow not completed 2",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionFdeEncryption,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:   ActionAddAptProxy,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:   ActionReboot,
									Status: tink.WorkflowStateRunning,
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("3/3: %s", WorkflowStepToStatusDetail[ActionReboot]),
		},
		{
			name: "Multiple actions - workflow not completed 3",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionFdeEncryption,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:   ActionAddAptProxy,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:   ActionReboot,
									Status: tink.WorkflowStatePending,
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("3/3: %s", WorkflowStepToStatusDetail[ActionReboot]),
		},
		{
			name: "Unknown action",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionFdeEncryption,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:   "unknown-action",
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:   ActionReboot,
									Status: tink.WorkflowStatePending,
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("3/3: %s", WorkflowStepToStatusDetail[ActionReboot]),
		},
		{
			name: "Failed action",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionFdeEncryption,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:    ActionAddAptProxy,
									Status:  tink.WorkflowStateFailed,
									Message: "some message",
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("2/2: %s failed: some message", WorkflowStepToStatusDetail[ActionAddAptProxy]),
		},
		{
			name: "First action failed",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionFdeEncryption,
									Status: tink.WorkflowStateFailed,
								},
								{
									Name: ActionAddAptProxy,
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("1/2: %s failed", WorkflowStepToStatusDetail[ActionFdeEncryption]),
		},
		{
			name: "Failed action empty message",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionFdeEncryption,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:   ActionAddAptProxy,
									Status: tink.WorkflowStateFailed,
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("2/2: %s failed", WorkflowStepToStatusDetail[ActionAddAptProxy]),
		},
		{
			name: "Timed out action",
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   ActionFdeEncryption,
									Status: tink.WorkflowStateSuccess,
								},
								{
									Name:    ActionAddAptProxy,
									Status:  tink.WorkflowStateTimeout,
									Message: "some message",
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("2/2: %s timeout", WorkflowStepToStatusDetail[ActionAddAptProxy]),
		},
	}
	for action, detail := range WorkflowStepToStatusDetail {
		tests = append(tests, struct {
			name string
			args args
			want string
		}{
			name: fmt.Sprintf("SingleAction_%s_Success", action),
			args: args{
				workflow: &tink.Workflow{Status: tink.WorkflowStatus{
					Tasks: []tink.Task{
						{
							Actions: []tink.Action{
								{
									Name:   action,
									Status: tink.WorkflowStateSuccess,
								},
							},
						},
					},
				}},
			},
			want: fmt.Sprintf("1/1: %s", detail),
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GenerateStatusDetailFromWorkflowState(tt.args.workflow),
				"GenerateStatusDetailFromWorkflowState(%v)", tt.args.workflow)
		})
	}
}
