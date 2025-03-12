// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// NOTICE: This file has been modified by Intel Corporation.
// Original file can be found at https://github.com/tinkerbell/actions.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"qemu-nbd-img2disk/image"

	dd "github.com/intel-tiber/infra-onboarding/tinker-actions/pkg/drive_detection"

	"github.com/cenkalti/backoff"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

const (
	defaultRetry            = true
	defaultRetryDuration    = 10
	defaultProgressInterval = 3
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer done()

	disk := os.Getenv("DEST_DISK")

	img := os.Getenv("IMG_URL")
	compressedEnv := os.Getenv("COMPRESSED")
	retryEnabled := os.Getenv("RETRY_ENABLED")
	retryDuration := os.Getenv("RETRY_DURATION_MINUTES")
	progressInterval := os.Getenv("PROGRESS_INTERVAL_SECONDS")
	textLogging := os.Getenv("TEXT_LOGGING")

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))
	if tlog, _ := strconv.ParseBool(textLogging); tlog {
		w := os.Stderr
		log = slog.New(tint.NewHandler(w, &tint.Options{
			NoColor: !isatty.IsTerminal(w.Fd()),
		}))
	}

	// Check if a string is empty
	if len(disk) == 0 {
		// Get a list of drives
		drives, err := dd.GetDrives()
		if err != nil {
			log.Error("Get Drive Error", "err", err)
			os.Exit(1)
		}
		detectedDisk, err := dd.DriveDetection(drives)
		if err != nil {
			log.Error("Drive detection Error", "err", err)
			os.Exit(1)
		}
		log.Info("Detected drive:", "detectedDisk", detectedDisk)
		disk = detectedDisk
	} else {
		log.Info("Drive provided by the user:", "disk", disk)
	}

	log.Info("QEMU-NBD-IMAGE2DISK - Cloud image streamer")

	if img == "" {
		log.Error("IMG_URL is required", "image", img)
		os.Exit(1)
	}

	if disk == "" {
		log.Error("DEST_DISK is required", "disk", disk)
		os.Exit(1)
	}

	u, err := url.Parse(img)
	if err != nil {
		log.Error("error parsing image URL (IMG_URL)", "err", err, "image", img)
		os.Exit(1)
	}

	expectedSHA256 := os.Getenv("SHA256")
	// if SHA256 env variable provided as input,compare the expected SHA256 with img_url SHA256
	if len(expectedSHA256) == 0 {
		fmt.Printf("-----SHA256 not provided proceeding without checksum check \n")
	}

	// We can ignore the error and default compressed to false.
	cmp, _ := strconv.ParseBool(compressedEnv)
	re, er := strconv.ParseBool(retryEnabled)
	if er != nil {
		re = defaultRetry
	}
	pi, err := strconv.Atoi(progressInterval)
	if err != nil {
		pi = defaultProgressInterval
	}

	// convert progress interval to duration in seconds
	interval := time.Duration(pi) * time.Second

	operation := func() error {
		if err := image.Write(ctx, log, u.String(), disk, cmp, interval); err != nil {
			return fmt.Errorf("error writing image to disk: %w", err)
		}
		return nil
	}

	if re {
		log.Info("retrying of qemu-nbd-image2disk is enabled")
		boff := backoff.NewExponentialBackOff()
		rd, err := strconv.Atoi(retryDuration)
		if err != nil {
			rd = defaultRetryDuration
			if retryDuration == "" {
				log.Info(fmt.Sprintf("no retry duration specified, using %v minutes for retry duration", rd))
			} else {
				log.Info(fmt.Sprintf("error converting retry duration to integer, using %v minutes for retry duration", rd), "err", err)
			}
		}
		boff.MaxElapsedTime = time.Duration(rd) * time.Minute
		bctx := backoff.WithContext(boff, ctx)
		retryNotifier := func(err error, duration time.Duration) {
			log.Error("retrying qemu-nbd-image2disk", "err", err, "duration", duration)

		}
		// try to write the image to disk with exponential backoff for 10 minutes
		if err := backoff.RetryNotify(operation, bctx, retryNotifier); err != nil {
			log.Error("error writing image to disk", "err", err, "image", img, "disk", disk)
			os.Exit(1)
		}
	} else {
		// try to write the image to disk without retry
		if err := operation(); err != nil {
			log.Error("error writing image to disk", "err", err, "image", img, "disk", disk)
			os.Exit(1)
		}
	}

	log.Info("Successfully wrote image to disk", "image", img, "disk", disk)
}
