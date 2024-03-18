// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package common

import "flag"

var (
	FlagDisableCredentialsManagement = flag.Bool("disableCredentialsManagement", false,
		"Disables credentials management for edge nodes. Should only be used for testing")
	FlagEnableDeviceInitialization = flag.Bool("enableDeviceInitialization", true,
		"Enables the device initialization phase during provisioning")
	FlagRVEnabled = flag.Bool("rvenabled", false, "Set to true if you have enabled rv")
)
