// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package auth

import (
	"context"
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
)

const (
	defaultTimeout = 3 * time.Second
)

// AuthService implements the authorization service to create or revoke EN credentials.
// Remember to call auth.Init() at the very beginning.
type AuthService interface { //nolint:revive // Need this interface name for more readable.
	// CreateCredentialsWithUUID creates EN credentials based on UUID.
	// The credentials can be further used by edge node agents.
	CreateCredentialsWithUUID(ctx context.Context, uuid string) (string, string, error)
	// GetCredentialsByUUID obtains EN credentials based on UUID.
	GetCredentialsByUUID(ctx context.Context, uuid string) (string, string, error)
	// RevokeCredentialsByUUID revokes EN credentials based on UUID.
	RevokeCredentialsByUUID(ctx context.Context, uuid string) error

	// Logout closes the session with authorization service.
	// Should always be invoked after all operations in a session are done.
	Logout(ctx context.Context)
}

// Init bootstraps the auth service library. Must be called after secrets.Init().
func Init() error {
	if *common.FlagDisableCredentialsManagement {
		zlog.Warn().Msgf("disableCredentialsManagement flag is set to false, " +
			"skip auth initialization")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Note that this function only creates auth service and logs out immediately.
	// The assumption is that AuthServiceFactory will perform all necessary initializations.
	authService, err := AuthServiceFactory(ctx)
	if err != nil {
		return err
	}
	defer authService.Logout(ctx)

	return nil
}
