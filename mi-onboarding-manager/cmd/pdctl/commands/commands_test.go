/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func Test_printUsage(t *testing.T) {
	type args struct {
		c    *cobra.Command
		args []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				c: &cobra.Command{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := printUsage(tt.args.c, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("printUsage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
