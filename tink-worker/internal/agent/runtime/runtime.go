// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package runtime contains runtime implementations that can execute workflow actions. They are
// responsible for extracting workflow failure reason and messages from the the action
// file system at the following locations:
//
//	/tinkerbell/failure-reason
//	/tinkerbell/failure-message
package runtime
