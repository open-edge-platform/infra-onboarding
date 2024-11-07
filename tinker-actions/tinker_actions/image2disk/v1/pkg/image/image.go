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

package image

// This package handles the pulling and management of images

import (
	"compress/bzip2"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"crypto/sha256"
        "encoding/hex"
        "bytes"

	"github.com/dustin/go-humanize"
	"github.com/klauspost/compress/zstd"
	log "github.com/sirupsen/logrus"
	"github.com/ulikunitz/xz"
	"golang.org/x/sys/unix"
)

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	return n, nil
}

func tickerProgress(byteCounter uint64) {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(byteCounter))
}

// Write will pull an image and write it to local storage device
// with compress set to true it will use gzip compression to expand the data before
// writing to an underlying device.
func Write(sourceImage, destinationDevice string, compressed bool) error {
	req, err := http.NewRequestWithContext(context.TODO(), "GET", sourceImage, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

        // if SHA256 env variable provided as input,compare the expected SHA256 with img url SHA256
        expectedSHA256 := os.Getenv("SHA256")
        if len(expectedSHA256) !=0 {
                fmt.Printf("INSIDE THE SHA FUNCTION ---\n")
                // Verify expected SHA256 with sourceImage downloaded SHA256
                bodyBytes, err := ioutil.ReadAll(resp.Body)
                if err != nil {
                        log.Fatal(err)
                }
                hash := sha256.Sum256(bodyBytes)
                actualSHA256 := hex.EncodeToString(hash[:])

                // convert the checksum to hexa
                //actualSHA256 := hex.EncodeToString(checksum)
                // compare the actualSHA256 with expectedSHA256
                // if matches write the image to disk,else discard
                if actualSHA256 != expectedSHA256 {
                        fmt.Printf("Mismatch SHA256 for actualSHA256 & expectedSHA256 ---\n")
                        log.Infof("expectedSHA256 : [%s] ", expectedSHA256)
                        log.Infof("actualSHA256 : [%s] ", actualSHA256)
                        log.Fatal("SHA256 MISMATCH")
                }
                fmt.Printf(" SHA256 MATCHED ---\n")
                resp.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
        }

	if resp.StatusCode > 300 {
		// Customise response for the 404 to make degugging simpler
		if resp.StatusCode == 404 {
			return fmt.Errorf("%s not found", sourceImage)
		}
		return fmt.Errorf("%s", resp.Status)
	}

	var out io.Reader

	fileOut, err := os.OpenFile(destinationDevice, os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer fileOut.Close()

	if !compressed {
		// Without compression send raw output
		out = resp.Body
	} else {
		// Find compression algorithm based upon extension
		decompressor, err := findDecompressor(sourceImage, resp.Body)
		if err != nil {
			return err
		}
		defer decompressor.Close()
		out = decompressor
	}

	log.Infof("Beginning write of image [%s] to disk [%s]", filepath.Base(sourceImage), destinationDevice)
	// Create our progress reporter and pass it to be used alongside our writer
	ticker := time.NewTicker(500 * time.Millisecond)
	counter := &WriteCounter{}

	go func() {
		for ; true; <-ticker.C {
			tickerProgress(counter.Total)
		}
	}()
	if _, err = io.Copy(fileOut, io.TeeReader(out, counter)); err != nil {
		ticker.Stop()
		return err
	}

	count, err := io.Copy(fileOut, out)
	if err != nil {
		ticker.Stop()
		return fmt.Errorf("error writing %d bytes to disk [%s] -> %w", count, destinationDevice, err)
	}
	fmt.Printf("\n")

	ticker.Stop()

	// Do the equivalent of partprobe on the device
	if err := fileOut.Sync(); err != nil {
		return fmt.Errorf("failed to sync the block device")
	}

	if err := unix.IoctlSetInt(int(fileOut.Fd()), unix.BLKRRPART, 0); err != nil {
		// Ignore errors since it may be a partition, but log in case it's helpful
		log.Errorf("error re-probing the partitions for the specified device: %v", err)
	}

	return nil
}

func findDecompressor(imageURL string, r io.Reader) (io.ReadCloser, error) {
	switch filepath.Ext(imageURL) {
	case ".bzip2":
		return ioutil.NopCloser(bzip2.NewReader(r)), nil
	case ".gz":
		reader, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("[ERROR] New gzip reader: %w", err)
		}
		return reader, nil
	case ".xz":
		reader, err := xz.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("[ERROR] New xz reader: %w", err)
		}
		return ioutil.NopCloser(reader), nil
	case ".zs":
		reader, err := zstd.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("[ERROR] New zs reader: %w", err)
		}
		return reader.IOReadCloser(), nil
	}

	return nil, fmt.Errorf("unknown compression suffix [%s]", filepath.Ext(imageURL))
}
