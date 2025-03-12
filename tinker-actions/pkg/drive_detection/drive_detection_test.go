// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package drive_detection

import (
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
			"/dev/sda",
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
