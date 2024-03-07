/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"bytes"
	"context"
	"fmt"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"net/http"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/auth"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
)

func SendFileToOwner(ownerIP, ownerSvcPort, guid, clientidsuffix, key string) error {
	attempts := 0
retry:
	url := fmt.Sprintf("http://%s:%s/api/v1/owner/resource?filename=%s_%s", ownerIP, ownerSvcPort, guid, clientidsuffix)

	zlog.Info().Msgf("Sending file %s_%s to FDO owner", guid, clientidsuffix)

	resp, err := http.Post(url, "text/plain", bytes.NewReader([]byte(key)))
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("")
		return inv_errors.Errorf("Failed to send file to FDO owner via resource API")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if attempts < 2 { // Check if we have retries left, considering 0 as the first attempt.
			attempts++
			zlog.Debug().Msgf("Retrying to send file %s_%s to FDO owner, attempts=%d",
				guid, clientidsuffix, attempts)
			goto retry
		}

		return inv_errors.Errorf("Failed to reach FDO Owner API with status code %v, attempts=%d",
			resp.StatusCode, attempts)
	}

	zlog.Debug().Msgf("File %s_%s successfully sent to FDO owner via resource API", guid, clientidsuffix)
	return nil
}

func ExecuteSVI(ownerIP, ownerSvcPort, clientidsuffix, clientsecretsuffix string) error {
	url := fmt.Sprintf("http://%s:%s/api/v1/owner/svi", ownerIP, ownerSvcPort)
	payload := fmt.Sprintf(`[{"filedesc" : "client_id","resource" : "$(guid)_%s"},{"filedesc" : "client_secret","resource" : "$(guid)_%s"}]`, clientidsuffix, clientsecretsuffix)

	zlog.Info().Msgf("Executing SVI for %s and %s", clientidsuffix, clientsecretsuffix)

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(payload))
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("")
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		zlog.MiSec().MiErr(err).Msg("")
		return err
	}
	defer resp.Body.Close()

	zlog.Debug().Msgf("Owner SVI API executed successfully for %s and %s", clientidsuffix, clientsecretsuffix)

	return nil
}

func InitializeDeviceSecretData(deviceInfo utils.DeviceInfo) error {
	// Save details to the database
	err := SendFileToOwner(deviceInfo.FdoOwnerDNS, deviceInfo.FdoOwnerPort, deviceInfo.FdoGUID, "client_id", deviceInfo.ClientID)
	if err != nil {
		return fmt.Errorf("error sending file to owner: %v", err)
	}
	err = SendFileToOwner(deviceInfo.FdoOwnerDNS, deviceInfo.FdoOwnerPort, deviceInfo.FdoGUID, "client_secret", deviceInfo.ClientSecret)
	if err != nil {
		return fmt.Errorf("error sending file to owner: %v", err)
	}
	//doing svi for secret Transfer
	err = ExecuteSVI(deviceInfo.FdoOwnerDNS, deviceInfo.FdoOwnerPort, "client_id", "client_secret")
	if err != nil {
		return fmt.Errorf("error doing svi for clientid: %v", err)
	}

	// Log completion message
	zlog.Debug().Msgf("InitializeDeviceSecretData completed for host %s (IP: %s)",
		deviceInfo.GUID, deviceInfo.HwIP)

	return nil
}

func GetClientData(deviceGUID string) (string, string, error) {
	ctx := context.Background()
	authService, err := auth.AuthServiceFactory(ctx)
	if err != nil {
		return "", "", err
	}
	defer authService.Logout(ctx)

	clientID, clientSecret, credsErr := authService.CreateCredentialsWithUUID(ctx, deviceGUID)
	if credsErr != nil {
		return "", "", credsErr
	}

	return clientID, clientSecret, nil
}
