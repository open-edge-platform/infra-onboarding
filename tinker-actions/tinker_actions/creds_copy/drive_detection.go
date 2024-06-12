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
	"os"
	"sort"
	"strings"
	"fmt"
    "path/filepath"

	log "github.com/sirupsen/logrus"
)

type LsblkOutput struct {
	BlockDevices []DriveInfo `json:"blockdevices`
}
type DriveInfo struct {
	Name string `json:"name"`
	Size uint64 `json:"size"`
	Type string `json:"type"`
	Tran string `json:"tran"`
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
//	log.Infof("Filtered Drives:")
//	for _, drive := range filteredDrives {
//		log.Infof("%+v\n", drive)
//	}
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

//	output := getDisksjson()
	lsblkOutput := getDisksjson()


	drives = lsblkOutput.BlockDevices
	return drives, nil
}

func filterDrives(drives []DriveInfo) []DriveInfo {
	var filteredDrives []DriveInfo

	for _, drive := range drives {
		// Check conditions: size not zero and type is "disk" and transport type is not "usb"
		if drive.Size != 0 && drive.Type == "disk" && drive.Tran != "usb" {
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

//-------------- lsblk immitation below
func getDisksjson() LsblkOutput {
    var disks []DriveInfo

    disks = getDriveInfos()
    lsblkOutput := LsblkOutput{BlockDevices: disks}

    return lsblkOutput
}

func getDriveInfos() []DriveInfo {
    var disks []DriveInfo

    blockDir := "/sys/block/"
    entries, err := os.ReadDir(blockDir)
    if err != nil {
        fmt.Println("Error:", err)
        return disks
    }

    for _, entry := range entries {
        entryPath := filepath.Join(blockDir, entry.Name())
        fileInfo, err := os.Lstat(entryPath)
        if err != nil {
            fmt.Println("Error:", err)
            continue
        }
        // Check if entry is a symbolic link
        if fileInfo.Mode()&os.ModeSymlink != 0 {
            // If it's a symbolic link, resolve it to get the actual name
            deviceName := resolveSymLink(entryPath)
            disk := getDiskInfo(deviceName)
            disks = append(disks, disk)
        }
    }

    return disks
}

func resolveSymLink(symLinkPath string) string {
    target, err := os.Readlink(symLinkPath)
    if err != nil {
        fmt.Println("Error resolving symbolic link:", err)
        return ""
    }
    return filepath.Base(target)
}

func getDiskInfo(name string) DriveInfo {
    var disk DriveInfo

    disk.Name = name
    disk.Type = "disk"

    sizePath := filepath.Join("/sys/block", name, "size")
    size, err := readUint64FromFile(sizePath)
    if err != nil {
        fmt.Println("Error:", err)
        return disk
    }
    // The size is in 512-byte sectors, so we convert it to bytes
    disk.Size = size * 512

    transportPath := filepath.Join("/sys/block", name, "device", "uevent")
    transport, err := getDeviceType(transportPath)
    if err != nil {
        fmt.Println("Error:", err)
        return disk
    }
    disk.Tran = transport

    return disk
}

func readUint64FromFile(filePath string) (uint64, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return 0, err
    }
    defer file.Close()

    var value uint64
    _, err = fmt.Fscanf(file, "%d", &value)
    if err != nil {
        return 0, err
    }

    return value, nil
}

// getDeviceType reads the uevent file at the given path
// and returns the device type based on the MODALIAS entry.
func getDeviceType(ueventPath string) (string, error) {

	if _, err := os.Stat(ueventPath); os.IsNotExist(err) {
		// File does not exist, return empty string
		return "", nil
	}

	data, err := os.ReadFile(ueventPath)
	if err != nil {
		return "", fmt.Errorf("error reading uevent file: %v", err)
	}

	// Convert the data to a string and split into lines
	lines := strings.Split(string(data), "\n")
	var modalias string
	for _, line := range lines {
		if strings.HasPrefix(line, "MODALIAS=") {
			modalias = strings.TrimPrefix(line, "MODALIAS=")
			break
		}
	}

	if modalias == "" {
		return "", fmt.Errorf("MODALIAS not found in uevent file")
	}

	// Extract the prefix from the MODALIAS
	prefix := strings.Split(modalias, ":")[0]

	return prefix, nil
}