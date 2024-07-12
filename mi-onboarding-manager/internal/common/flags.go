// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package common

import "flag"

var (
	FlagEnableDeviceInitialization = flag.Bool("enableDeviceInitialization", true,
		"Enables the device initialization phase during provisioning")
	FlagRVEnabled = flag.Bool("rvenabled", false, "Set to true if you have enabled rv")
)
