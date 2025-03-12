// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

module github.com/tinkerbell/hub/actions/writefile/v1

go 1.23.0

require (
	github.com/intel-tiber/infra-onboarding/tinker-actions/pkg/drive_detection v0.0.0-20250311120014-fe933a9e83cb
	github.com/sirupsen/logrus v1.9.3
)

replace github.com/intel-tiber/infra-onboarding/tinker-actions/pkg/drive_detection => ../../pkg/drive_detection

require (
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
	gopkg.in/yaml.v3 v3.0.0-20220521103104-8f96da9f5d5e // indirect
)
