// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package drive_detection

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDriveDetection(t *testing.T) {
	// Mocking mockGetDrives function
	testCases := []struct {
		mockDrives   []DriveInfo
		expectedDisk string
	}{

		{[]DriveInfo{
			{Name: "sda", Size: 1024, Type: "disk"},
			{Name: "nvme0n1", Size: 51, Type: "disk"},
			{Name: "sdb", Size: 256, Type: "disk"},
		},
			"/dev/sdb",
		},
		{[]DriveInfo{
			{Name: "sda", Size: 1024, Type: "disk", Tran: "sata"},
			{Name: "nvme0n1", Size: 51, Type: "disk"},
			{Name: "sdb", Size: 256, Type: "disk", Tran: "usb"},
		},
			"/dev/sdb",
		},
		{[]DriveInfo{
			{Name: "sda", Size: 1024, Type: "disk"},
			{Name: "sdb", Size: 256, Type: "disk"},
		},
			"/dev/sdb",
		},
		{[]DriveInfo{
			{Name: "sda", Size: 256, Type: "disk"},
			{Name: "sdb", Size: 256, Type: "disk"},
		},
			"/dev/sda",
		},
		{[]DriveInfo{
			{Name: "nvme0n1", Size: 256, Type: "disk"},
			{Name: "nvme0n2", Size: 256, Type: "disk"},
		},
			"/dev/nvme0n1",
		},
		{[]DriveInfo{
			{Name: "nvme0n1", Size: 256, Type: "disk"},
			{Name: "sdc", Size: 2560, Type: "disk"},
			{Name: "nvme0n2", Size: 256, Type: "disk"},
		},
			"/dev/sdc",
		},
	}
	for _, tc := range testCases {
		response, err := DriveDetection(tc.mockDrives)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if response != tc.expectedDisk {
			t.Errorf("Expected drive: %s, got: %s", tc.expectedDisk, response)
		}

	}
}
func TestDriveDetection_NoValidDrives(t *testing.T) {
	// Mocking GetDrives function to return an empty list
	drives := []DriveInfo{}

	// Test DriveDetection function when no valid drives are found
	disk, err := DriveDetection(drives)

	expectedError := "No valid drives found."
	if err == nil || err.Error() != expectedError {
		t.Errorf("Expected error: %s, got: %v", expectedError, err)
	}

	if disk != "" {
		t.Errorf("Expected empty drive, got: %s", disk)
	}
}

func TestDriveDetection_SingleValidDrive(t *testing.T) {
	// Mocking GetDrives function to return an empty list
	drives := []DriveInfo{
		{Name: "sdc", Size: 2560, Type: "disk"},
		{Name: "sdc1", Size: 2560, Type: "part"},
	}

	// Test DriveDetection function when no valid drives are found
	disk, err := DriveDetection(drives)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if disk != "/dev/sdc" {
		t.Errorf("Expected /dev/sdc, got: %s", disk)
	}
}

func TestFindRootPartitionForDisks(t *testing.T) {
	t.Run("RunWithRealDeviceOrFakeEnv", func(t *testing.T) {
		_, err := FindRootPartitionForDisk("/dev/thisdoesnotexist")
		if err == nil {
			t.Fatal("Expected error due to lsblk failure, but got nil")
		}
		t.Logf("Received expected error: %v", err)
	})
	t.Run("FallbackToExt4", func(t *testing.T) {
		tempDir := t.TempDir()
		fakeLsblk := filepath.Join(tempDir, "lsblk")
		mockOutput := `{
		"blockdevices": [
			{
				"name": "sda1",
				"type": "part",
				"fstype": "ext4",
				"label": "rootfs",
				"partlabel": "Linux filesystem"
			}
		]
	}`
		script := fmt.Sprintf(`#!/bin/sh
echo '%s'`, mockOutput)
		if err := os.WriteFile(fakeLsblk, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to write fake lsblk: %v", err)
		}
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)
		os.Setenv("PATH", tempDir+":"+originalPath)
		result, err := FindRootPartitionForDisk("dummy-disk")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		expected := "/dev/sda1"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("NoRootfsOrExt4Partition", func(t *testing.T) {
		tempDir := t.TempDir()
		fakeLsblk := filepath.Join(tempDir, "lsblk")
		mockOutput := `{
		"blockdevices": [
			{
				"name": "sdb1",
				"type": "part",
				"fstype": "ext4",
				"label": "data",
				"partlabel": ""
			}
		]
	}`
		script := fmt.Sprintf(`#!/bin/sh
echo '%s'`, mockOutput)
		if err := os.WriteFile(fakeLsblk, []byte(script), 0755); err != nil {
			t.Fatalf("Failed to write fake lsblk: %v", err)
		}
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)
		os.Setenv("PATH", tempDir+":"+originalPath)
		result, err := FindRootPartitionForDisk("dummy-disk")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expected := "/dev/sdb1"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
}

func TestGetDrives(t *testing.T) {
	tests := []struct {
		name    string
		want    []DriveInfo
		wantErr bool
	}{
		{
			name:    "Get Drives",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDrives()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDrives() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDrives() = %v, want %v", got, tt.want)
			}
		})
	}
}
