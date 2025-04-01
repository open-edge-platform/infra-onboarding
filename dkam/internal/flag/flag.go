// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package flag

import "flag"

var LegacyMode = flag.Bool("legacyMode", false,
	"Set to enable curation of legacy Installer script for mutable OSes that will be pushed to PV.")
