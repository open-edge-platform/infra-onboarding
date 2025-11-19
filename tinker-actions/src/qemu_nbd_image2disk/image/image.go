// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package image

// This package handles the pulling and management of images

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
	"golang.org/x/sys/unix"
)

// Write will pull an image and write it to network boot device (nbd) using qemu-nbd
// before writing to an underlying device.
func Write(ctx context.Context, log *slog.Logger, sourceImage, destinationDevice string, compressed bool, progressInterval time.Duration, tlsCaCert []byte) error {
	// Create HTTP client with custom TLS configuration if CA cert is provided
	client := http.DefaultClient
	if len(tlsCaCert) > 0 {
		err, valid := validate_cert(log, tlsCaCert)
		if err != nil {
			log.Error("Failed to validate CA certificate", "error", err)
			return fmt.Errorf("failed to validate CA certificate: %w", err)
		}
		if !valid {
			log.Error("Invalid CA certificate")
			return fmt.Errorf("invalid CA certificate")
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(tlsCaCert) {
			log.Error("Failed to append CA cert to pool - certificate may be corrupted or invalid")
			return fmt.Errorf("failed to append CA cert to pool: certificate is not valid PEM format or is corrupted")
		}

		log.Info("Successfully added CA certificate to trust pool")

		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
			Proxy: http.ProxyFromEnvironment,
		}
		client = &http.Client{Transport: transport}
		log.Info("HTTP client configured with custom TLS settings")
	} else {
		log.Info("No TLS CA certificate provided, using default HTTP client")
	}

	// Create and execute an HTTP GET request to download the image
	req, err := http.NewRequestWithContext(ctx, "GET", sourceImage, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
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
	hashReader := io.TeeReader(resp.Body, hash)

	var dataReader io.Reader = hashReader

	// If compressed, wrap with appropriate decompressor
	if compressed {
		log.Info("Decompressing image", "format", filepath.Ext(sourceImage))
		decompressor, err := createDecompressor(sourceImage, hashReader)
		if err != nil {
			return fmt.Errorf("failed to create decompressor: %w", err)
		}
		defer decompressor.Close()
		dataReader = decompressor
	}

	// Copy the image to tmp file and simultaneously write to the hash
	_, err = io.Copy(tmpFile, dataReader)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("failed to save the image to temp file: %v", err)
	}
	tmpFile.Close()
	log.Info("Successfully saved image to tmpFile")

	// Run 'ls -lh /tmp' and print stdout and stderr
	cmdLsTmp := exec.Command("ls", "-lh", "/tmp")
	var lsTmpOut, lsTmpErr bytes.Buffer
	cmdLsTmp.Stdout = &lsTmpOut
	cmdLsTmp.Stderr = &lsTmpErr
	if err := cmdLsTmp.Run(); err != nil {
		return fmt.Errorf("failed to run ls -lh /tmp: %v\nstdout:%s\nstderr:\n%s", err, lsTmpOut.String(), lsTmpErr.String())
	}
	log.Info("ls -lh /tmp output", "stdout", lsTmpOut.String(), "stderr", lsTmpErr.String())

	// Compute the SHA-256 checksum
	hashSum := hash.Sum(nil)
	actualSHA256 := hex.EncodeToString(hashSum)
	log.Info(fmt.Sprintf("SHA-256 hash of the downloaded file: %s", actualSHA256))

	expectedSHA256 := os.Getenv("SHA256")
	// if SHA256 env variable provided as input, compare the expected SHA256 with img_url SHA256
	if len(expectedSHA256) != 0 && actualSHA256 != expectedSHA256 {
		log.Info("-----Mismatch SHA256 for actualSHA256 & expectedSHA256 ---\n")
		log.Info(fmt.Sprintf("expectedSHA256 : [%s] ", expectedSHA256))
		log.Info(fmt.Sprintf("actualSHA256 : [%s] ", actualSHA256))
		log.Error("------SHA256 MISMATCH---------")
		return fmt.Errorf("image SHA-256 hash mismatch")
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

	// Run 'ls -ld /var/lock' to show directory details
	cmdLs := exec.Command("ls", "-lrth", "/dev/nbd0")
	var lsOut, lsErr bytes.Buffer
	cmdLs.Stdout = &lsOut
	cmdLs.Stderr = &lsErr
	if err := cmdLs.Run(); err != nil {
		return fmt.Errorf("failed to run ls: %v\nstdout:%s\nstderr:\n%s", err, lsOut.String(), lsErr.String())
	}
	log.Info("ls output", "stdout", lsOut.String(), "stderr", lsErr.String())

	// Attach the qcow2 image as a network block device
	var outBuf, errBuf bytes.Buffer
	nbdDevice := "/dev/nbd0"
	cmdNbd := exec.Command("qemu-nbd", "--connect="+nbdDevice, tmpFile.Name())
	cmdNbd.Stdout = &outBuf
	cmdNbd.Stderr = &errBuf
	if err := cmdNbd.Run(); err != nil {
		return fmt.Errorf("network block device attach failed: %v\nstdout:%s\nstderr:\n%s", err, outBuf.String(), errBuf.String())
	}
	log.Info("qemu-nbd connect output", "stdout", outBuf.String(), "stderr", errBuf.String())
	defer exec.Command("qemu-nbd", "--disconnect", nbdDevice).Run()
	log.Info("Successfully attached qcow2 image as network block device")

	// Run 'lsblk' in a loop until output contains "nbd0p1"
	var lsblkOut, lsblkErr bytes.Buffer
	for {
		cmdLsblk := exec.Command("lsblk")
		cmdLsblk.Stdout = &lsblkOut
		cmdLsblk.Stderr = &lsblkErr
		if err := cmdLsblk.Run(); err != nil {
			return fmt.Errorf("failed to run lsblk: %v\nstdout:%s\nstderr:\n%s", err, lsblkOut.String(), lsblkErr.String())
		}
		log.Info("lsblk output", "stdout", lsblkOut.String(), "stderr", lsblkErr.String())
		if bytes.Contains(lsblkOut.Bytes(), []byte("nbd0p1")) {
			log.Info("Found nbd0p1 in lsblk output, proceeding with installation")
			break
		}
		time.Sleep(2 * time.Second)
		log.Info("Waiting for nbd0p1 to appear in lsblk output, retrying...")
		lsblkOut.Reset()
		lsblkErr.Reset()
	}

	cmdLs1 := exec.Command("ls", "/var/lock")
	var lsOut1, lsErr1 bytes.Buffer
	cmdLs1.Stdout = &lsOut1
	cmdLs1.Stderr = &lsErr1
	if err := cmdLs1.Run(); err != nil {
		return fmt.Errorf("failed to run ls: %v\nstdout:%s\nstderr:\n%s", err, lsOut1.String(), lsErr1.String())
	}
	log.Info("ls output", "stdout", lsOut1.String(), "stderr", lsErr1.String())

	// Install the OS to the disk using DD
	cmdDD := exec.Command("dd", "if="+nbdDevice, "of="+destinationDevice, "bs=4M")
	cmdDD.Stdout = os.Stdout
	cmdDD.Stderr = os.Stderr

	if err := cmdDD.Run(); err != nil {
		return fmt.Errorf("failed to write image to disk: %v", err)
	}
	log.Info(fmt.Sprintf("Successfully installed  cloud image on %s", destinationDevice))

	// Rerun 'lsblk' and print stdout and stderr
	cmdLsblk2 := exec.Command("lsblk")
	var lsblkOut2, lsblkErr2 bytes.Buffer
	cmdLsblk2.Stdout = &lsblkOut2
	cmdLsblk2.Stderr = &lsblkErr2
	if err := cmdLsblk2.Run(); err != nil {
		return fmt.Errorf("failed to rerun lsblk: %v\nstdout:%s\nstderr:\n%s", err, lsblkOut2.String(), lsblkErr2.String())
	}
	log.Info("lsblk output (rerun)", "stdout", lsblkOut2.String(), "stderr", lsblkErr2.String())

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

func validate_cert(log *slog.Logger, tlsCaCert []byte) (error, bool) {
	certStr := string(tlsCaCert)
	// Validate that the certificate data looks like PEM format
	if !bytes.Contains(tlsCaCert, []byte("-----BEGIN CERTIFICATE-----")) {
		return fmt.Errorf("invalid CA certificate: missing PEM header '-----BEGIN CERTIFICATE-----'"), false
	}
	if !bytes.Contains(tlsCaCert, []byte("-----END CERTIFICATE-----")) {
		return fmt.Errorf("invalid CA certificate: missing PEM footer '-----END CERTIFICATE-----'"), false
	}
	block, rest := pem.Decode(tlsCaCert)
	if block == nil {
		log.Error("Failed to decode CA certificate as PEM", "cert", certStr)
		log.Debug("Certificate data", "data", certStr)
		log.Debug("Successfully decoded PEM block", "remainingBytes", len(rest))
		// Parse the certificate to validate it before adding to pool
		return fmt.Errorf("invalid CA certificate: failed to parse X.509 certificate"), false
	}
	// Verify it's actually a certificate
	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Error("Failed to parse certificate from PEM block", "error", err)
		return fmt.Errorf("invalid CA certificate: failed to parse X.509 certificate: %w", err), false
	}
	log.Debug("Successfully validated certificate structure")

	return nil, true
}

// createDecompressor returns a ReadCloser that decompresses data based on file extension
func createDecompressor(imagePath string, reader io.Reader) (io.ReadCloser, error) {
	ext := filepath.Ext(imagePath)

	switch ext {
	case ".bz2", ".bzip2":
		return io.NopCloser(bzip2.NewReader(reader)), nil

	case ".gz":
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzReader, nil

	case ".xz":
		xzReader, err := xz.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create xz reader: %w", err)
		}
		return io.NopCloser(xzReader), nil

	case ".zs", ".zst":
		zstdReader, err := zstd.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd reader: %w", err)
		}
		return zstdReader.IOReadCloser(), nil

	default:
		return nil, fmt.Errorf("unsupported compression format: %s", ext)
	}
}
