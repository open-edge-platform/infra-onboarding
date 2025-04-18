// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package drive_detection

import (
	"encoding/json"
	"os/exec"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

type DriveDetectionLsblkOutput struct {
	BlockDevices []DriveInfo `json:"blockdevices"`
}
type RootPartitionDetectionLsblkOutput struct {
	BlockDevices []PartitionInfo `json:"blockdevices"`
}
type DriveInfo struct {
	Name        string `json:"name"`
	Size        uint64 `json:"size"`
	Type        string `json:"type"`
	Tran        string `json:"tran"`
	IsRemovable bool   `json:"rm"`
}
type PartitionInfo struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	FileSystemType string `json:"fstype"`
	Label          string `json:"label"`
	PartitionLabel string `json:"partlabel"`
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

	var lsblkOutput DriveDetectionLsblkOutput
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
		if drive.Size != 0 && drive.Type == "disk" && !drive.IsRemovable &&
			!(strings.HasPrefix(drive.Name, "mmcblk") && strings.Contains(drive.Name, "boot")) {

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

// findRootPartitionForDisk finds the root partition for a given disk by
// detecting rootfs tag or finding the first partition with ext4 filesystem type
func FindRootPartitionForDisk(disk string) (string, error) {
	// If filesystem type is not provided, default to ext4
	const rootPartitionTag = "rootfs"
	const defaultFsType = "ext4"

	var partitions []PartitionInfo

	// Command to list all partitions for the input disk with filesystem type and label using lsblk
	cmd := exec.Command("lsblk", "--output", "NAME,TYPE,FSTYPE,LABEL,PARTLABEL", "-ln", "--json", disk)

	// Run the command and capture the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Infof("Error running command:%s", err)
		return "", err
	}

	var lsblkOutput RootPartitionDetectionLsblkOutput
	err = json.Unmarshal([]byte(output), &lsblkOutput)
	if err != nil {
		log.Infof("Error running command:%s", err)
		return "", err
	}

	partitions = lsblkOutput.BlockDevices
	// Find the root partition with rootfs tag
	for _, partition := range partitions {
		if partition.Type == "part" &&
			(strings.Contains(partition.PartitionLabel, rootPartitionTag) || strings.Contains(partition.Label, rootPartitionTag)) {
			log.Infof("Match found with label %s: %+v", rootPartitionTag, partition)
			rootPartition := "/dev/" + partition.Name
			return rootPartition, nil
		}
	}

	log.Infof("could not find match with label %s for disk %s", rootPartitionTag, disk)

	// If root partition is not found with rootfs tag, find the first partition with ext4 filesystem type
	for _, partition := range partitions {
		if partition.Type == "part" && partition.FileSystemType == defaultFsType {
			log.Infof("Match found with fstype %s: %+v", defaultFsType, partition)
			rootPartition := "/dev/" + partition.Name
			return rootPartition, nil
		}
	}
	return "", &CustomError{Message: "root partition not found for disk" + disk}
}
