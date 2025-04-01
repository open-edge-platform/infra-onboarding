/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/auth"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	logging "github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	loggerName = "ClientSecret"
)

var zlogCliSecret = logging.GetLogger(loggerName)

func FetchClientSecret(ctx context.Context, tenantID, uuid string) (clientID, clientSecret string, err error) {
	authService, err := auth.AuthServiceFactory(ctx)
	if err != nil {
		return "", "", err
	}
	defer authService.Logout(ctx)

	clientID, clientSecret, err = authService.GetCredentialsByUUID(ctx, tenantID, uuid)
	if err != nil && inv_errors.IsNotFound(err) {
		return authService.CreateCredentialsWithUUID(ctx, tenantID, uuid)
	}

	if err != nil {
		zlogCliSecret.InfraSec().InfraErr(err).Msgf("")
		// some other error that may need retry
		zlogCliSecret.Debug().Msgf("Failed to check if EN credentials for host %s exist.", uuid)
		return "", "", inv_errors.Errorf("Failed to check if EN credentials for host exist.")
	}

	zlogCliSecret.Debug().Msgf("EN credentials for host %s already exists.", uuid)

	return clientID, clientSecret, nil
}
