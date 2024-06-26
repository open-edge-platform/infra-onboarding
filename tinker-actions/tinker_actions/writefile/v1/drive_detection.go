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
	"encoding/json"
	"os/exec"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

type LsblkOutput struct {
	BlockDevices []DriveInfo `json:"blockdevices`
}
type DriveInfo struct {
	Name        string `json:"name"`
	Size        uint64 `json:"size"`
	Type        string `json:"type"`
	Tran        string `json:"tran"`
	IsRemovable bool   `json:"rm"`
}

// Get disk type of Drive based on drive name
func (d DriveInfo) getDiskType() string {
	if strings.HasPrefix(d.Name, "nvme") {
		return "nvme"
	} else if strings.HasPrefix(d.Name, "sd") {
		return "sd"
	} else {
		return "NA"
	}
}

func DriveDetection(drives []DriveInfo) (string, error) {

	// Filter out devices with size zero or type not equal to "disk"
	filteredDrives := filterDrives(drives)
	// Display the filtered list of drives
	log.Infof("Filtered Drives:")
	for _, drive := range filteredDrives {
		log.Infof("%+v\n", drive)
	}
	// Sort drives based on the defined criteria
	sort.Sort(byPriorityAndSize(filteredDrives))

	log.Infof("Filtered and Sorted Drives:")
	for _, drive := range filteredDrives {
		log.Infof("%+v\n", drive)
	}

	//detected drive
	if len(filteredDrives) >= 1 {
		disk := "/dev/" + filteredDrives[0].Name
		if len(filteredDrives) == 1 {
			log.Warnln("************************")
			log.Warnf("Only ONE DISK detected (%s). There will be NO support for persistent volume available for apps and some edge node functionalities will NOT work!", disk)
			log.Warnln("************************")
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
	cmd := exec.Command("lsblk", "--output", "NAME,TYPE,SIZE,TRAN,RM", "-bldn", "--json")

	// Run the command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Infof("Error running command:%s", err)
		return nil, err
	}

	var lsblkOutput LsblkOutput
	err = json.Unmarshal([]byte(output), &lsblkOutput)
	if err != nil {
		log.Infof("Error running command:%s", err)
		return nil, err
	}

	drives = lsblkOutput.BlockDevices
	return drives, nil
}

func filterDrives(drives []DriveInfo) []DriveInfo {
	var filteredDrives []DriveInfo

	for _, drive := range drives {
		// Check conditions: size not zero and type is "disk" and is non-removable
		if drive.Size != 0 && drive.Type == "disk" && !drive.IsRemovable {
			filteredDrives = append(filteredDrives, drive)
		}
	}

	return filteredDrives
}

type byPriorityAndSize []DriveInfo

var driveTypeRanking = map[string]int{ // Priority: sd >> nvme >> others
	"sd":   0,
	"nvme": 1,
	"NA":   2,
}

func (a byPriorityAndSize) Len() int      { return len(a) }
func (a byPriorityAndSize) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byPriorityAndSize) Less(i, j int) bool {
	// If Priority is same
	if driveTypeRanking[a[i].getDiskType()] == driveTypeRanking[a[j].getDiskType()] {
		// If Size is same
		if a[i].Size == a[j].Size {
			// Choose the drive which comes first in alphabetical order
			return a[i].Name < a[j].Name
		}

		// Choose the drive with lower size
		return a[i].Size < a[j].Size

	}
	// Choose the drive with higher priority
	return driveTypeRanking[a[i].getDiskType()] < driveTypeRanking[a[j].getDiskType()]

}
