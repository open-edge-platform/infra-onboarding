// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"fmt"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
)

var workflowStepToStatusDetail = map[string]string{
	ActionSecureBootStatusFlagRead:   "Verifying Secure Boot settings match",
	ActionStreamUbuntuImage:          "Streaming OS image",
	ActionCopySecrets:                "Copying secrets",
	ActionGrowPartitionInstallScript: "Growing partition",
	ActionInstallOpenssl:             "Installing OpenSSL",
	ActionCreateUser:                 "Creating user",
	ActionEnableSSH:                  "Enabling SSH",
	ActionDisableApparmor:            "Disabling apparmor",
	ActionInstallScriptDownload:      "Downloading installation scripts",
	ActionInstallScript:              "Installing packages",
	ActionInstallScriptEnable:        "Enabling system services",
	ActionNetplan:                    "Enabling network",
	ActionNetplanConfigure:           "Configuring network settings",
	ActionGrowPartitionService:       "Starting grow partition service",
	ActionGrowPartitionServiceEnable: "Enabling grow partition service",
	ActionNetplanService:             "Starting netplan update service",
	ActionNetplanServiceEnable:       "Enabling netplan update service",
	ActionEfibootset:                 "Setting boot option",
	ActionFdeEncryption:              "Setting FDE encryption",
	ActionReboot:                     "Rebooting",
	ActionCustomerID:                 "Setting customer ID",
	ActionCopyENSecrets:              "Copying secrets",
	ActionStoringAlpine:              "Storing Alpine",
	ActionRunFDO:                     "Running FDO",
	ActionAddEnvProxy:                "Configuring system proxy settings",
	ActionAddAptProxy:                "Configuring APT proxy settings",
	ActionAddDNSNamespace:            "Configuring DNS settings",
	ActionCreateSecretsDirectory:     "Creating secrets directory",
	ActionWriteClientID:              "Saving client ID",
	ActionWriteClientSecret:          "Saving client secret",
	ActionWriteHostname:              "Setting host name",
	ActionWriteEtcHosts:              "Setting into /etc/hosts",
	ActionCreateCustomerIDDirectory:  "Create directory in EN for Customer ID",
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

		statusDetail, ok := workflowStepToStatusDetail[action.Name]
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
