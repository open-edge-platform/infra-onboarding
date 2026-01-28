// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
//
//nolint:testpackage // Keeping the test in the same package due to dependencies on unexported fields.
package signing

import (
	"fmt"
	"os"
	"testing"
)

func Test_copyFile(t *testing.T) {
	type args struct {
		src string
		dst string
	}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("error while getting directory : ", err)
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				src: "",
			},
			wantErr: true,
		},
		{
			name: "Test Case1",
			args: args{
				src: wd,
				dst: "",
			},
			wantErr: true,
		},
		{
			name: "Test Case 2",
			args: args{
				src: wd,
				dst: wd + "dummy",
			},
			wantErr: true,
		},
	}
	defer func() {
		_ = os.Remove(wd + "dummy")
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := copyFile(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("copyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_copyDir(t *testing.T) {
	type args struct {
		src string
		dst string
	}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("error while getting directory : ", err)
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				dst: "",
			},
			wantErr: true,
		},
		{
			name: "Test Case",
			args: args{
				dst: wd + "dummy",
			},
			wantErr: true,
		},
	}
	defer func() {
		_ = os.RemoveAll(wd + "dummy")
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := copyDir(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("copyDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
