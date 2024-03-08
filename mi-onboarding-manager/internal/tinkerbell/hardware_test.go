// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"reflect"
	"testing"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
)

func TestNewHardware(t *testing.T) {
	type args struct {
		name    string
		ns      string
		id      string
		device  string
		ip      string
		gateway string
	}
	tests := []struct {
		name string
		args args
		want *tink.Hardware
	}{
		{
			name: "Test Case 1",
			args: args{},
			want: &tink.Hardware{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHardware(tt.args.name, tt.args.ns, tt.args.id, tt.args.device,
				tt.args.ip, tt.args.gateway); reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHardware() = %v, want %v", got, tt.want)
			}
		})
	}
}
