// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package drive_detection

import (
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

func TestFindRootPartitionForDisk(t *testing.T) {
	type args struct {
		disk string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test case",
			args: args{
				disk: "/dev/sda",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindRootPartitionForDisk(tt.args.disk)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindRootPartitionForDisk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FindRootPartitionForDisk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDrives(t *testing.T) {
	tests := []struct {
		name    string
		want    []DriveInfo
		wantErr bool
	}{
		{
			name: "Get Drives",
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
