// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package secrets

import (
	"context"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/secrets"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/common"
)

const (
	onboardingCredentialsSecretName = "host-manager-m2m-client-secret"
)

var (
	secretClientID     string
	secretClientSecret string
)

var zlog = logging.GetLogger("SecretService")

var inst = &secretService{}

// SecretService implements the interaction with the secrets storage (e.g., Vault).
type SecretService interface {
	// Init initializes the SecretService.
	// It should always be invoked at the very beginning, before other methods are used.
	Init(ctx context.Context) error
	// GetClientID obtains the `client_id` secret (in the UUID format) from the SecretService.
	GetClientID()
	// GetClientSecret obtains the `client_secret` secret from the SecretService.
	GetClientSecret()
}

type secretService struct {
}

func Init(ctx context.Context) error {
	return inst.init(ctx)
}

func GetClientID() string {
	return secretClientID
}

func GetClientSecret() string {
	return secretClientSecret
}

func (ss *secretService) init(ctx context.Context) error {
	if *common.FlagDisableCredentialsManagement {
		zlog.Warn().Msgf("disableCredentialsManagement flag is set to false, " +
			"skip secrets initialization")
		return nil
	}

	vaultS, err := secrets.SecretServiceFactory(ctx)
	if err != nil {
		return err
	}
	defer vaultS.Logout(ctx)

	credentials, err := vaultS.ReadSecret(ctx, onboardingCredentialsSecretName)
	if err != nil {
		return err
	}

	dataMap, ok := credentials["data"].(map[string]interface{})
	if !ok {
		err = inv_errors.Errorf("Cannot read credentials data from Vault secret")
		zlog.MiSec().Err(err).Msg("")
		return err
	}

	_clientID, exists := dataMap["client_id"]
	if !exists {
		err = inv_errors.Errorf("Failed to get client_id from secrets service")
		zlog.MiSec().Err(err).Msg("")
		return err
	}
	clientID, ok := _clientID.(string)
	if !ok {
		err = inv_errors.Errorf("Wrong format of client_id read from Vault, expected string, got %T", _clientID)
		zlog.MiSec().Err(err).Msg("")
		return err
	}
	secretClientID = clientID

	_clientSecret, exists := dataMap["client_secret"]
	if !exists {
		err = inv_errors.Errorf("Failed to get client_id from secrets service")
		zlog.MiSec().Err(err).Msg("")
		return err
	}
	clientSecret, ok := _clientSecret.(string)
	if !ok {
		err = inv_errors.Errorf("Wrong format of client_secret read from Vault, expected string, got %T", _clientSecret)
		zlog.MiSec().Err(err).Msg("")
		return err
	}
	secretClientSecret = clientSecret

	zlog.MiSec().Debug().Msgf("Secrets successfully initialized")

	return nil
}
