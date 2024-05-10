// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package signing

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// func TestBuildSignIpxe(t *testing.T) {
// 	type args struct {
// 		scriptPath string
// 		dnsName    string
// 	}
// 	wd, _ := os.Getwd()
// 	result := strings.Replace(wd, "signing", "script", -1)
// 	res := filepath.Join(result, "latest")
// 	dir := config.PVC
// 	os.MkdirAll(dir, 0755)
// 	if err := os.MkdirAll(filepath.Dir(res), 0755); err != nil {
// 		t.Fatalf("Failed to create directory: %v", err)
// 	}
// 	CopyFile(result+"/chain.ipxe", res)
// 	tests := []struct {
// 		name    string
// 		args    args
// 		want    bool
// 		wantErr bool
// 	}{
// 		{
// 			name: "Test Case",
// 			args: args{
// 				scriptPath: result,
// 				dnsName:    "",
// 			},
// 			want:    true,
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := BuildSignIpxe(config.PVC, tt.args.scriptPath, tt.args.dnsName)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("BuildSignIpxe() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if got != tt.want {
// 				t.Errorf("BuildSignIpxe() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// 	defer func() {
// 		CopyFile(res, result+"/chain.ipxe")
// 		os.Remove(res)
// 		os.Remove(dir)
// 	}()
// }

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
