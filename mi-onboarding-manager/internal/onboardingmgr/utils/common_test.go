/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaSlculateRootF(t *testing.T) {
	// Test case 1: imageType is "bkc" and diskDev ends with a numeric digit
	partition := CalculateRootFS("bkc", "sda1")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")

	// Test case 2: imageType is "ms" and diskDev ends with a numeric digit
	partition = CalculateRootFS("ms", "nvme0n1p2")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")

	// Test case 3: imageType is "bkc" and diskDev does not end with a numeric digit
	partition = CalculateRootFS("bkc", "sdb")
	assert.Equal(t, "1", partition, "Expected partition '1'")

	// Test case 4: imageType is  "ms" and diskDev ends with a numeric digit
	partition = CalculateRootFS("other", "nvme0n1p3")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")
}
