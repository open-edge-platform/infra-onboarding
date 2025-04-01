// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

module github.com/tinkerbell/hub/actions/writefile/v1

go 1.24.1

require (
	github.com/open-edge-platform/infra-onboarding/tinker-actions/pkg/drive_detection v0.0.0-20250324105403-f8fa27a1b024
	github.com/sirupsen/logrus v1.9.3
)

replace github.com/open-edge-platform/infra-onboarding/tinker-actions/pkg/drive_detection => ../../pkg/drive_detection

require (
	golang.org/x/sys v0.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20220521103104-8f96da9f5d5e // indirect
)
