// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package auth

import "context"

// AuthService implements the authorization service to create or revoke EN credentials.
type AuthService interface {
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
