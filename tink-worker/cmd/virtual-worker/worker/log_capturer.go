// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"

	"github.com/tinkerbell/tink/cmd/tink-worker/worker"
)

type emptyLogger struct{}

func (l *emptyLogger) CaptureLogs(context.Context, string) {}

// NewEmptyLogCapturer returns an no-op log capturer.
func NewEmptyLogCapturer() worker.LogCapturer {
	return &emptyLogger{}
}
