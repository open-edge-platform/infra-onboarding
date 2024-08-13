// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package signing

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		return err
	}
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func Test_copyFile(t *testing.T) {
	type args struct {
		src string
		dst string
	}
	wd, _ := os.Getwd()
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := copyFile(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("copyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		os.Remove(wd + "dummy")
	}()
}

func Test_copyDir(t *testing.T) {
	type args struct {
		src string
		dst string
	}
	wd, _ := os.Getwd()
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := copyDir(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("copyDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	defer func() {
		os.RemoveAll(wd + "dummy")
	}()
}

func Test_contains(t *testing.T) {
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "failure",
			args: args{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.args.slice, tt.args.s); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
