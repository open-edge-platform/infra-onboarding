// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tinkerbell

import (
	"fmt"
)

func GetWorkflowName(uuid string) string {
	return fmt.Sprintf("workflow-%s", uuid)
}

func GetTinkHardwareName(uuid string) string {
	return fmt.Sprintf("machine-%s", uuid)
}
