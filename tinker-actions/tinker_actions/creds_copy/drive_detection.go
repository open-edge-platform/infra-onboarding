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

package main

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type DriveInfo struct {
	Name string
	Size uint64
	Type string
}

func DriveDetection(drives []DriveInfo) (string, error) {
	// Sort drives based on the defined criteria
	sort.Sort(bySizeAndPriority(drives))

	// // Display the sorted list of drives
	log.Infof("Sorted Drives:")
	for _, drive := range drives {
		log.Infof("Name: %s, Size: %d Bytes, Type: %s", drive.Name, drive.Size, drive.Type)
	}

	// Filter out devices with size zero or type not equal to "disk"
	filteredDrives := filterDrives(drives)
	// Print the result
	log.Infof("Filtered and Sorted Drives:")
	for _, drive := range filteredDrives {
		log.Infof("Name: %s, Size: %d Bytes, Type: %s", drive.Name, drive.Size, drive.Type)
	}

	//detected drive
	if len(filteredDrives) >= 1 {
		disk := "/dev/" + filteredDrives[0].Name
		// Check if the drive name starts with "/dev/nvme"
		if strings.HasPrefix(disk, "/dev/nvme") {
			disk = disk + "p1"
		} else {
			disk = disk + "1"
		}
		return disk, nil
	} else {
		return "", &CustomError{Message: "No valid drives found."}
	}

}

// CustomError is a custom error type that satisfies the error interface.
type CustomError struct {
	Message string
}

// Error returns the error message.
func (e *CustomError) Error() string {
	return e.Message
}

func GetDrives() ([]DriveInfo, error) {
	var drives []DriveInfo

	// Command to list all connected storage devices with sizes using lsblk
	cmd := exec.Command("lsblk", "--output", "NAME,TYPE,SIZE", "-b")

	// Run the command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Infof("Error running command:%s", err)
		return nil, err
	}

	output_after_removing_traling_whitespaces := strings.TrimSpace(string(output))
	// Convert the combined output to a list of strings
	outputLines := strings.Split(string(output_after_removing_traling_whitespaces), "\n")
	// Print each line in the list
	log.Infof("Command Output:")
	for _, line := range outputLines[1:] {
		line_after_removing_extra_spaces := removeExtraSpaces(line)
		line := strings.Split(string(line_after_removing_extra_spaces), " ")

		deviceName := line[0]
		deviceType := line[1]
		diskSizeInString := line[2]
		diskSize, err := strconv.ParseUint(string(diskSizeInString), 10, 64)

		if err != nil {
			log.Infof("Error converting string to uint64:%s", err)
			return nil, err
		}
		drives = append(drives, DriveInfo{
			Name: string(deviceName),
			Size: diskSize,
			Type: string(deviceType),
		})
	}

	return drives, nil
}

func removeExtraSpaces(input string) string {
	// Split the string into words
	words := strings.Fields(input)

	// Join the words back together with a single space
	result := strings.Join(words, " ")

	return result
}

func filterDrives(drives []DriveInfo) []DriveInfo {
	var filteredDrives []DriveInfo

	for _, drive := range drives {
		// Check conditions: size not zero and type is "disk"
		if drive.Size != 0 && drive.Type == "disk" {
			filteredDrives = append(filteredDrives, drive)
		}
	}

	return filteredDrives
}

type bySizeAndPriority []DriveInfo

func (a bySizeAndPriority) Len() int      { return len(a) }
func (a bySizeAndPriority) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a bySizeAndPriority) Less(i, j int) bool {
	if a[i].Size == a[j].Size {
		// If sizes are equal
		if strings.HasPrefix(a[i].Name, "nvme") && !strings.HasPrefix(a[j].Name, "nvme") {
			// If only i has nvme prefix
			return true
		} else if !strings.HasPrefix(a[i].Name, "nvme") && strings.HasPrefix(a[j].Name, "nvme") {
			// If only j has nvme prefix
			return false
		} else if strings.HasPrefix(a[i].Name, "nvme") && strings.HasPrefix(a[j].Name, "nvme") {
			// If both have nvme prefix, choose chronologically
			return a[i].Name < a[j].Name
		} else {
			// If none have nvme prefix, choose chronologically
			return a[i].Name < a[j].Name
		}
	}
	// Choose the drive with smallest size
	return a[i].Size < a[j].Size
}
