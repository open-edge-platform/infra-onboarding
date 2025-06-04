// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package event

import (
	"fmt"
)

// IncompatibleError indicates an event was received that.
type IncompatibleError struct {
	Event Event
}

func (e IncompatibleError) Error() string {
	return fmt.Sprintf("incompatible event: %v", e.Event.GetName())
}
