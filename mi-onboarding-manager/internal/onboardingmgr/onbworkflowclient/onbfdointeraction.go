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

	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
)

func SendFileToOwner(ownerIP, ownerSvcPort, guid, clientidsuffix, key string) error {
	attempts := 0
retry:
	url := fmt.Sprintf("http://%s:%s/api/v1/owner/resource?filename=%s_%s", ownerIP, ownerSvcPort, guid, clientidsuffix)

	zlog.Info().Msgf("Sending file %s_%s to FDO owner", guid, clientidsuffix)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBufferString(key))
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

	if resp.StatusCode != http.StatusOK {
		if attempts < retryAttempts { // Check if we have retries left, considering 0 as the first attempt.
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
	payload := fmt.Sprintf(
		`[{"filedesc" : "client_id","resource" : "$(guid)_%s"},{"filedesc" : "client_secret","resource" : "$(guid)_%s"}]`,
		clientidsuffix, clientsecretsuffix)

	zlog.Info().Msgf("Executing SVI for %s and %s", clientidsuffix, clientsecretsuffix)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBufferString(payload))
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
