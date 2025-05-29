// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"fmt"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
)

var WorkflowStepToStatusDetail = map[string]string{
	ActionEraseNonRemovableDisk:    "Erasing data from all non-removable disks",
	ActionSecureBootStatusFlagRead: "Verifying Secure Boot settings",
	ActionInstallScriptDownload:    "Downloading installation scripts",
	ActionStreamOSImage:            "Streaming OS image",
	ActionInstallScript:            "Installing packages",
	ActionInstallScriptEnable:      "Enabling system services",
	ActionEfibootset:               "Setting boot option",
	ActionFdeEncryption:            "Setting FDE encryption",
	ActionSecurityFeatures:         "Enabling OS security features",
	ActionReboot:                   "Rebooting",
	ActionAddAptProxy:              "Configuring APT proxy settings",
	ActionSystemdNetworkOptimize:   "Applying optimized Systemd network settings",
	ActionDisableSnapdOptimize:     "Disabling Snapd service for optimization",
	ActionKernelupgrade:            "Upgrading kernel",
	ActionCloudInitInstall:         "Installing cloud-init",
	ActionCloudinitDsidentity:      "Setting up cloud-init",
}

func GenerateStatusDetailFromWorkflowState(workflow *tink.Workflow) string {
	if workflow == nil {
		// no status detail if workflow doesn't exist yet
		return ""
	}

	if workflow.Status.State == tink.WorkflowStateSuccess {
		// no need to return intermediate state for successful workflow
		return ""
	}

	workflowTasks := workflow.Status.Tasks

	if len(workflowTasks) == 0 {
		zlog.Debug().Msgf("No tasks defined for workflow %s, returning empty status detail", workflow.Name)
		return ""
	}

	// NOTE: we assume there is always 1 task for a workflow (see template_data.go).
	// It may be changed in the future.
	actions := workflowTasks[0].Actions

	totalActions := len(actions)

	if totalActions == 0 {
		zlog.Warn().Msgf("No actions defined for workflow %s, invalid workflow", workflow.Name)
		return ""
	}

	return prepareStatusDetails(totalActions, actions)
}

func prepareStatusDetails(totalActions int, actions []tink.Action) string {
	currActionNumber := 1
	message := ""
	for i, action := range actions {
		if action.Name == "" {
			zlog.Warn().Msgf("A Tink action with empty name, invalid workflow")
			return ""
		}

		// find first non-success action, unless it's last action
		if action.Status == tink.WorkflowStateSuccess && i != len(actions)-1 {
			continue
		}

		statusDetail, ok := WorkflowStepToStatusDetail[action.Name]
		if !ok {
			// it should never happen, but we set a raw action name just in case
			statusDetail = action.Name
		}

		message = statusDetail

		if action.Status == tink.WorkflowStateFailed {
			message = fmt.Sprintf("%s failed", statusDetail)
			if action.Message != "" {
				message += fmt.Sprintf(": %s", action.Message)
			}
		}

		if action.Status == tink.WorkflowStateTimeout {
			message = fmt.Sprintf("%s timeout", statusDetail)
		}

		currActionNumber = i + 1
		break
	}

	return fmt.Sprintf("%d/%d: %s", currActionNumber, totalActions, message)
}
