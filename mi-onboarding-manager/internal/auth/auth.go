// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package auth

import (
	"context"
	"time"
)

const (
	defaultTimeout = 3 * time.Second
)

// AuthService implements the authorization service to create or revoke EN credentials.
// Remember to call auth.Init() at the very beginning.
type AuthService interface { //nolint:revive // Need this interface name for more readable.
	// Init initializes the auth service. It should only be called once at the very beginning.
	Init(ctx context.Context) error
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	authService, err := AuthServiceFactory(ctx)
	if err != nil {
		return err
	}
	defer authService.Logout(ctx)

	return authService.Init(ctx)
}
