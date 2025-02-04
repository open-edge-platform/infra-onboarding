// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"fmt"
)

func GetProdWorkflowName(uuid string) string {
	return fmt.Sprintf("workflow-%s-prod", uuid)
}

func GetProdTemplateName(uuid string) string {
	return fmt.Sprintf("template-%s-prod", uuid)
}

func GetTinkHardwareName(uuid string) string {
	return fmt.Sprintf("machine-%s", uuid)
}
