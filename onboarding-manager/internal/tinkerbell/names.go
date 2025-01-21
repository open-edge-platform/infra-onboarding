// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"fmt"
	"strings"
)

func GetRebootWorkflowName(uuid string) string {
	return fmt.Sprintf("reboot-workflow-%s", uuid)
}

func GetProdWorkflowName(uuid string) string {
	return fmt.Sprintf("workflow-%s-prod", uuid)
}

func GetRebootTemplateName(uuid string) string {
	return fmt.Sprintf("reboot-%s", uuid)
}

func GetProdTemplateName(imageType, uuid string) string {
	imgTypeStr := strings.ReplaceAll(imageType, "_", "-")
	return fmt.Sprintf("%s-%s-prod", imgTypeStr, uuid)
}

func GetTinkHardwareName(uuid string) string {
	return fmt.Sprintf("machine-%s", uuid)
}
