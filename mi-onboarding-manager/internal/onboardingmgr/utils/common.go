/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/auth"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	logging "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
)

const (
	loggerName           = "utils"
	FileModeReadWriteAll = 0o777
)

var zlog = logging.GetLogger(loggerName)
var (
	once            sync.Once
	file            *os.File
	errLogTimeStamp error
)

func CalculateRootFS(imageType, diskDev string) string {
	rootFSPartNo := "1"

	if imageType == "bkc" {
		rootFSPartNo = "1"
	}

	// Use regular expression to check if diskDev ends with a numeric digit
	match, err := regexp.MatchString(".*[0-9]$", diskDev)
	if err != nil {
		return rootFSPartNo
	}
	if match {
		return fmt.Sprintf("p%s", rootFSPartNo)
	}

	return rootFSPartNo
}

// ReplaceHostIP finds %host_ip% in the url string and replaces it with ip.
func ReplaceHostIP(url, ip string) string {
	// Define the regular expression pattern to match #host_ip
	re := regexp.MustCompile(`%host_ip%`)
	return re.ReplaceAllString(url, ip)
}

// TODO : Will scale it in future accordingly.
func IsValidOSURLFormat(osURL string) bool {
	expectedSuffix := ".raw.gz" // Checks if the OS URL is in the expected format
	return strings.HasSuffix(osURL, expectedSuffix)
}

// Init initializes the timestamp logger and opens the file for writing timestamps.
func Init(filename string) {
	once.Do(func() {
		file, errLogTimeStamp = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, FileModeReadWriteAll)
		if errLogTimeStamp != nil {
			zlog.MiSec().MiErr(errLogTimeStamp).Msgf("failed to open timestamp log file")
		}
	})
}

// TimeStamp writes a timestamped message to the log file.
func TimeStamp(message string) {
	timestamp := time.Now().Format(time.RFC3339)
	_, err := fmt.Fprintf(file, "%s: %s\n", timestamp, message)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("failed to write to timestamp log file")
	}
}

// Close closes the log file.
func Close() {
	if file != nil {
		err := file.Close()
		if err != nil {
			zlog.MiSec().MiErr(err).Msgf("failed to close timestamp log file")
		}
	}
}

func FetchClientSecret(ctx context.Context, uuid string) (string, string, error) {
	authService, err := auth.AuthServiceFactory(ctx)
	if err != nil {
		return "", "", err
	}
	defer authService.Logout(ctx)

	clientID, clientSecret, err := authService.GetCredentialsByUUID(ctx, uuid)
	if err != nil && inv_errors.IsNotFound(err) {
		return authService.CreateCredentialsWithUUID(ctx, uuid)
	}

	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("")
		// some other error that may need retry
		return "", "", inv_errors.Errorf("Failed to check if EN credentials for host %s exist.", uuid)
	}

	zlog.Debug().Msgf("EN credentials for host %s already exists.", uuid)

	return clientID, clientSecret, nil
}
