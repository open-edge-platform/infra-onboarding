// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package event

import "context"

// NoopRecorder retrieves a nooping fake recorder.
func NoopRecorder() *RecorderMock {
	return &RecorderMock{
		RecordEventFunc: func(context.Context, Event) error { return nil },
	}
}
