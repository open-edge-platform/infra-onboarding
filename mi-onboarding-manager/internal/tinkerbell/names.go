// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import "fmt"

func GetDIWorkflowName(uuid string) string {
	return fmt.Sprintf("di-workflow-%s", uuid)
}

func GetRebootWorkflowName(uuid string) string {
	return fmt.Sprintf("reboot-workflow-%s", uuid)
}

func GetProdWorkflowName(uuid string) string {
	return fmt.Sprintf("workflow-%s-prod", uuid)
}

func GetDITemplateName(uuid string) string {
	return fmt.Sprintf("fdodi-%s", uuid)
}

func GetRebootTemplateName(uuid string) string {
	return fmt.Sprintf("reboot-%s", uuid)
}

func GetProdTemplateName(imageType, uuid string) string {
	return fmt.Sprintf("%s-%s-prod", imageType, uuid)
}

func GetTinkHardwareName(uuid string) string {
	return fmt.Sprintf("machine-%s", uuid)
}
