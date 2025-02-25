// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import osv1 "github.com/intel/infra-core/inventory/v2/pkg/api/os/v1"

//nolint:revive // will be refactored soon
type CloudInitOptions struct {
	Mode   string
	OsType osv1.OsType
}
