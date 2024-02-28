/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/auth"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
)

func SendFileToOwner(ownerIP, ownerSvcPort, guid, clientidsuffix, key string) error {
	attempts := 0
retry:
	url := fmt.Sprintf("http://%s:%s/api/v1/owner/resource?filename=%s_%s", ownerIP, ownerSvcPort, guid, clientidsuffix)
	print(url)
	resp, err := http.Post(url, "text/plain", bytes.NewReader([]byte(key)))
	if err != nil {
		fmt.Println("Owner resource API failed:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if attempts < 2 { // Check if we have retries left, considering 0 as the first attempt.12
			attempts++
			fmt.Println("Owner resource API failed:", resp.StatusCode)
			goto retry
		}

		return fmt.Errorf("owner api failed with status code %d", resp.StatusCode)
	}

	fmt.Println("Owner resource API is success", resp.StatusCode)
	return nil
}

func ExecuteSVI(ownerIP, ownerSvcPort, clientidsuffix, clientsecretsuffix string) error {
	url := fmt.Sprintf("http://%s:%s/api/v1/owner/svi", ownerIP, ownerSvcPort)
	payload := fmt.Sprintf(`[{"filedesc" : "client_id","resource" : "$(guid)_%s"},{"filedesc" : "client_secret","resource" : "$(guid)_%s"}]`, clientidsuffix, clientsecretsuffix)

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(payload))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return err
	}
	fmt.Println(payload)
	print(url, payload)
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return err
	}
	defer resp.Body.Close()

	fmt.Println("Owner svi API is success", resp.Status)
	return nil
}

func InitializeDeviceSecretData(deviceInfo utils.DeviceInfo) error {

	// Get client data
	clientSecret, clientID, err := GetClientData(deviceInfo.GUID)
	if err != nil {
		return fmt.Errorf("error getting client data: %v", err)
	}
	// Save details to the database
	err = SendFileToOwner(deviceInfo.FdoOwnerDNS, deviceInfo.FdoOwnerPort, deviceInfo.FdoGUID, "client_id", clientID)
	if err != nil {
		return fmt.Errorf("error sending file to owner: %v", err)
	}
	err = SendFileToOwner(deviceInfo.FdoOwnerDNS, deviceInfo.FdoOwnerPort, deviceInfo.FdoGUID, "client_secret", clientSecret)
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

	clientSecret, clientID, credsErr := authService.CreateCredentialsWithUUID(ctx, deviceGUID)
	if credsErr != nil {
		return "", "", credsErr
	}

	return clientSecret.(string), clientID, nil
}
