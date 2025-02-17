// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	log "github.com/sirupsen/logrus"
)

const mountAction = "/mountAction"

func main() {
	fmt.Printf("creds_copy - Copy folder\n------------------------\n")

	// Parse the environment variables that are passed into the action
	blockDevice := os.Getenv("BLOCK_DEVICE")
	filesystemType := os.Getenv("FS_TYPE")
	//	isDestMount := os.Getenv("NO_MOUNT")
	if len(blockDevice) == 0 {
		// Get a list of drives
		drives, err := GetDrives()
		if err != nil {
			log.Error(err)
			return
		}
		detectedDisk, err := DriveDetection(drives)
		if err != nil {
			log.Error(err)
			return
		}
		log.Infof("Detected drive: [%s] ", detectedDisk)
		blockDevice, err = findRootPartitionForDisk(detectedDisk)
		if err != nil {
			log.Error(err)
			return
		}
		log.Infof("Drive detected by automation: [%s] ", blockDevice)
	} else {
		log.Infof("Drive provided by the user: [%s] ", blockDevice)
	}

	if blockDevice == "" {
		log.Fatalf("No Block Device speified with Environment Variable [BLOCK_DEVICE]")
	}

	// Create the /mountAction mountpoint (no folders exist previously in scratch container)
	err := os.Mkdir(mountAction, os.ModeDir)
	if err != nil {
		log.Fatalf("Error creating the action Mountpoint [%s]", mountAction)
	}

	// Mount the block device to the /mountAction point
	err = syscall.Mount(blockDevice, mountAction, filesystemType, 0, "")
	if err != nil {
		log.Fatalf("Mounting [%s] -> [%s] error [%v]", blockDevice, mountAction, err)
	}
	log.Infof("Mounted [%s] -> [%s]", blockDevice, mountAction)

	src := os.Getenv("DOCKER_SRC_PATH")
	if src == "" {
		src = "/dev/shm/boot"
	}

	dst_folder := os.Getenv("OS_DST_DIR")
	if dst_folder == "" {
		dst_folder = "/etc/ensp/node"
	}
	dest := filepath.Join(mountAction, dst_folder)

	err = CopyFolder(src, dest)
	if err != nil {
		fmt.Printf("Error copying folder: %v\n", err)
	} else {
		fmt.Println("Folder copied successfully.")
	}

	err = syscall.Unmount(mountAction, syscall.MNT_DETACH)
	if err != nil {
		fmt.Printf("Error unmounting: %v\n", err)
	} else {
		fmt.Println("Unmounted successfully.")
	}
}

func CopyFolder(src string, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			err := CopyFolder(sourcePath, destPath)
			if err != nil {
				return err
			}
		} else {
			err := Copy(sourcePath, destPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func Copy(src string, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}
	return nil
}
