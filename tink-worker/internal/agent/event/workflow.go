// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package event

const WorkflowRejectedName Name = "WorkflowRejected"

// WorkflowRejected is generated when a workflow is being rejected by the agent.
type WorkflowRejected struct {
	ID      string
	Message string
}

func (WorkflowRejected) GetName() Name {
	return WorkflowRejectedName
}

func (e WorkflowRejected) String() string {
	return e.Message
}
