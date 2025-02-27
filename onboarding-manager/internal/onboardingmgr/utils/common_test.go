/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/intel/infra-onboarding/onboarding-manager/internal/onboardingmgr/utils"
)

func TestCaSlculateRootF(t *testing.T) {
	// Test case 1: imageType is "bkc" and diskDev ends with a numeric digit
	partition := utils.CalculateRootFS("bkc", "sda1")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")

	// Test case 2: imageType is "ms" and diskDev ends with a numeric digit
	partition = utils.CalculateRootFS("ms", "nvme0n1p2")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")

	// Test case 3: imageType is "bkc" and diskDev does not end with a numeric digit
	partition = utils.CalculateRootFS("bkc", "sdb")
	assert.Equal(t, "1", partition, "Expected partition '1'")

	// Test case 4: imageType is  "ms" and diskDev ends with a numeric digit
	partition = utils.CalculateRootFS("other", "nvme0n1p3")
	assert.Equal(t, "p1", partition, "Expected partition 'p1'")
}

func TestReplaceHostIP(t *testing.T) {
	type args struct {
		url string
		ip  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Case",
			args: args{
				url: "",
				ip:  "",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.ReplaceHostIP(tt.args.url, tt.args.ip); got != tt.want {
				t.Errorf("ReplaceHostIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidOSURLFormat(t *testing.T) {
	type args struct {
		osURL string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test Case",
			args: args{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsValidOSURLFormat(tt.args.osURL); got != tt.want {
				t.Errorf("IsValidOSURLFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}
