// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package image

// This package handles the pulling and management of images

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

// Write will pull an image and write it to network boot device (nbd) using qemu-nbd
// before writing to an underlying device.
func Write(ctx context.Context, log *slog.Logger, sourceImage, destinationDevice string, compressed bool, progressInterval time.Duration) error {
	// Create and execute an HTTP GET request to download the image
	req, err := http.NewRequestWithContext(ctx, "GET", sourceImage, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download image from the URL: %v", err)
	}
	defer resp.Body.Close()
	log.Info("Successfully downloaded image")

	// Check if the response status code is 200
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image, HTTP status code: %d", resp.StatusCode)
	}

	// Create a temp file for storing the cloud image in qcow2 format
	tmpFile, err := os.CreateTemp("", "img-*.qcow2")
	if err != nil {
		return fmt.Errorf("temp file creation failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	log.Info("Successfully created empty temp file")

	// Create a SHA-256 hash object
	hash := sha256.New()

	// Copy the image to tmp file and simultaneously write to the hash
	_, err = io.Copy(io.MultiWriter(tmpFile, hash), resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save the image to temp file: %v", err)
	}
	tmpFile.Close()
	log.Info("Successfully saved image to tmpFile")

	// Compute the SHA-256 checksum
	hashSum := hash.Sum(nil)
	actualSHA256 := hex.EncodeToString(hashSum)
	log.Info(fmt.Sprintf("SHA-256 hash of the downloaded file: %s", actualSHA256))

	expectedSHA256 := os.Getenv("SHA256")
	// if SHA256 env variable provided as input, compare the expected SHA256 with img_url SHA256
	if len(expectedSHA256) != 0 && actualSHA256 != expectedSHA256 {
		log.Info(fmt.Sprintf("-----Mismatch SHA256 for actualSHA256 & expectedSHA256 ---\n"))
		log.Info(fmt.Sprintf("expectedSHA256 : [%s] ", expectedSHA256))
		log.Info(fmt.Sprintf("actualSHA256 : [%s] ", actualSHA256))
		log.Error("------SHA256 MISMATCH---------")
		return fmt.Errorf("Image SHA-256 hash mismatch")
	}
	log.Info(fmt.Sprintf("SHA-256 hash of the downloaded file: %s", actualSHA256))
	log.Info("Successfully verified SHA-256 checksum")

	// Load the nbd kernel module
	cmdModprobe := exec.Command("modprobe", "nbd")
	cmdModprobe.Stdout = os.Stdout
	cmdModprobe.Stderr = os.Stderr
	if err := cmdModprobe.Run(); err != nil {
		return fmt.Errorf("failed to load nbd kernel module: %v", err)
	}
	log.Info("Successfully loaded nbd kernel module")

	lockDir := "/run/lock/qemu-nbd-nbd0"
	if _, err := os.Stat(lockDir); err == nil {
		if err := os.RemoveAll(lockDir); err != nil {
			return fmt.Errorf("failed to remove existing lock directory %s: %v", lockDir, err)
		}
		log.Info(fmt.Sprintf("Removed existing lock directory: %s", lockDir))
	}
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return fmt.Errorf("failed to create lock directory %s: %v", lockDir, err)
	}
	log.Info(fmt.Sprintf("Created lock directory: %s", lockDir))
	// Attach the qcow2 image as a network block device
	var outBuf, errBuf bytes.Buffer
	nbdDevice := "/dev/nbd0"
	cmdNbd := exec.Command("qemu-nbd", "--connect="+nbdDevice, tmpFile.Name())
	cmdNbd.Stdout = &outBuf
	cmdNbd.Stderr = &errBuf
	if err := cmdNbd.Run(); err != nil {
		return fmt.Errorf("network block device attach failed: %v\nstdout:%s\nstderr:\n%s", err, outBuf.String(), errBuf.String())
	}
	defer exec.Command("qemu-nbd", "--disconnect", nbdDevice).Run()
	log.Info("Successfully attached qcow2 image as network block device")

	// Install the OS to the disk using DD
	cmdDD := exec.Command("dd", "if="+nbdDevice, "of="+destinationDevice, "bs=4M")
	cmdDD.Stdout = os.Stdout
	cmdDD.Stderr = os.Stderr

	if err := cmdDD.Run(); err != nil {
		return fmt.Errorf("failed to write image to disk: %v", err)
	}
	log.Info(fmt.Sprintf("Successfully installed  cloud image on %s", destinationDevice))

	// Run partition table re-probing
	file, err := os.OpenFile(destinationDevice, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open device %s: %v", destinationDevice, err)
	}
	defer file.Close()
	err = unix.IoctlSetInt(int(file.Fd()), unix.BLKRRPART, 0)
	if err != nil {
		return fmt.Errorf("failed to re-probe partitions on %s: %v", destinationDevice, err)
	}
	return nil
}
