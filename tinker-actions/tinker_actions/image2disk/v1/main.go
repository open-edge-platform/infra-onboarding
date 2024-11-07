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
	"fmt"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/tinkerbell/hub/actions/image2disk/v1/pkg/image"
)

func main() {
	fmt.Printf("IMAGE2DISK - Cloud image streamer\n------------------------\n")
	disk := os.Getenv("DEST_DISK")
	// Check if a string is empty
	if len(disk) == 0 {
		// Get a list of drives
		drives, err := image.GetDrives()
		if err != nil {
			log.Fatal(err)
		}
		detectedDisk, err := image.DriveDetection(drives)
			if err != nil {
			log.Fatal(err)
		}
		log.Infof("Detected drive: [%s] ", detectedDisk)
		disk = detectedDisk
	} else {
		log.Infof("Drive provided by the user: [%s] ", disk)
	}

	img := os.Getenv("IMG_URL")
	compressedEnv := os.Getenv("COMPRESSED")
	// We can ignore the error and default compressed to false.
	cmp, _ := strconv.ParseBool(compressedEnv)

	// Write the image to disk
	err := image.Write(img, disk, cmp)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Successfully written [%s] to [%s]", img, disk)
}
