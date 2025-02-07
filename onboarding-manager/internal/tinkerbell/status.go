// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"fmt"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
)

var WorkflowStepToStatusDetail = map[string]string{
	ActionEraseNonRemovableDisk:      "Erasing data from all non-removable disks",
	ActionSecureBootStatusFlagRead:   "Verifying Secure Boot settings",
	ActionStreamUbuntuImage:          "Streaming OS image",
	ActionCopySecrets:                "Copying secrets",
	ActionGrowPartitionInstallScript: "Growing partition",
	ActionInstallOpenssl:             "Installing OpenSSL",
	ActionCreateUser:                 "Creating user",
	ActionEnableSSH:                  "Enabling SSH",
	ActionDisableApparmor:            "Disabling AppArmor",
	ActionInstallScriptDownload:      "Downloading installation scripts",
	ActionInstallScript:              "Installing packages",
	ActionInstallScriptEnable:        "Enabling system services",
	ActionNetplan:                    "Enabling network",
	ActionNetplanConfigure:           "Configuring network settings",
	ActionGrowPartitionService:       "Starting grow partition service",
	ActionGrowPartitionServiceEnable: "Enabling grow partition service",
	ActionNetplanService:             "Starting Netplan update service",
	ActionNetplanServiceEnable:       "Enabling Netplan update service",
	ActionEfibootset:                 "Setting boot option",
	ActionFdeEncryption:              "Setting FDE encryption",
	ActionReboot:                     "Rebooting",
	ActionCopyENSecrets:              "Copying EN secrets",
	ActionStoringAlpine:              "Storing Alpine",
	ActionAddEnvProxy:                "Configuring system proxy settings",
	ActionAddAptProxy:                "Configuring APT proxy settings",
	ActionAddDNSNamespace:            "Configuring DNS settings",
	ActionCreateSecretsDirectory:     "Creating secrets directory",
	ActionWriteClientID:              "Saving client ID",
	ActionTenantID:                   "Saving tenant ID",
	ActionWriteClientSecret:          "Saving client secret",
	ActionWriteHostname:              "Setting hostname",
	ActionWriteEtcHosts:              "Adding entries to /etc/hosts",
	ActionSystemdNetworkOptimize:     "Applying optimized Systemd network settings",
	ActionDisableSnapdOptimize:       "Disabling Snapd service for optimization",
	ActionKernelupgrade:              "Setting kernel-upgrade",
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
