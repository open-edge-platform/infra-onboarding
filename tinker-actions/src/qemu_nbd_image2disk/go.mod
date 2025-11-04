// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

module qemu-nbd-img2disk

go 1.24.8

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/lmittmann/tint v1.0.7
	github.com/mattn/go-isatty v0.0.20
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/sys v0.31.0
)

require github.com/open-edge-platform/infra-onboarding/tinker-actions/pkg/drive_detection v0.0.0-20250324105403-f8fa27a1b024

replace github.com/open-edge-platform/infra-onboarding/tinker-actions/pkg/drive_detection => ../../pkg/drive_detection
