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
	"crypto/sha256"
        "encoding/hex"
        "io"
        "net/http"

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
	expectedSHA256 := os.Getenv("SHA256")

        // if SHA256 env variable provided as input,compare the expected SHA256 with img_url SHA256
        if len(expectedSHA256) !=0 {
                fmt.Printf("-----Calculating the SHA256 checksum for OS image ---\n")

                // get request for the image
                response, err := http.Get(img)
                if err != nil {
			log.Fatal(err)
                }
                defer response.Body.Close()
                hasher := sha256.New()

                // copy the response to hasher
                if _, err := io.Copy(hasher, response.Body); err != nil {
			log.Fatal(err)
                }
                // get the SHA-256 checksum in bytes
                checksum := hasher.Sum(nil)

                // convert the checksum to hexa
                actualSHA256 := hex.EncodeToString(checksum)

                // compare the actualSHA256 with expectedSHA256
                // if matches write the image to disk,else discard
                if actualSHA256 != expectedSHA256 {
                        fmt.Printf("-----Mismatch SHA256 for actualSHA256 & expectedSHA256 ---\n")
                        log.Infof("expectedSHA256 : [%s] ", expectedSHA256)
                        log.Infof("actualSHA256 : [%s] ", actualSHA256)
                        log.Fatal("------SHA256 MISMATCH---------")
                }
                fmt.Printf("-----SHA256 MATCHED & proceding for os installation ---\n")
	    }

	// We can ignore the error and default compressed to false.
	cmp, _ := strconv.ParseBool(compressedEnv)

	// Write the image to disk
	err := image.Write(img, disk, cmp)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Successfully written [%s] to [%s]", img, disk)
}
