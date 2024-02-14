/* #####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2024 Intel Corporation                                              #
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################*/

package image

import (
	"testing"
)

func TestDriveDetection(t *testing.T) {
	// Mocking mockGetDrives function
	drives := []DriveInfo{
		{Name: "sda", Size: 1024, Type: "disk"},
		{Name: "nvme0n1", Size: 512, Type: "disk"},
		{Name: "sdb", Size: 256, Type: "disk"},
	}

	// Test DriveDetection function
	disk, err := DriveDetection(drives)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedDisk := "/dev/sdb"
	if disk != expectedDisk {
		t.Errorf("Expected drive: %s, got: %s", expectedDisk, disk)
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
