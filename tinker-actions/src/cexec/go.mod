// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// NOTICE: This file has been modified by Intel Corporation.
// Original file can be found at https://github.com/tinkerbell/actions.
module cexec

go 1.24.9

require (
	github.com/open-edge-platform/infra-onboarding/tinker-actions/pkg/drive_detection v0.0.0
	github.com/peterbourgon/ff/v3 v3.4.0
)

replace github.com/open-edge-platform/infra-onboarding/tinker-actions/pkg/drive_detection => ../../pkg/drive_detection

require (
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/sys v0.31.0 // indirect
)
