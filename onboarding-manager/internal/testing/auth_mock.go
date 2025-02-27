// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/intel/infra-core/inventory/v2/pkg/auth"
	inv_errors "github.com/intel/infra-core/inventory/v2/pkg/errors"
)

const errIndex = 2

type authServiceMock struct {
	mock.Mock
}

func (a *authServiceMock) CreateCredentialsWithUUID(
	ctx context.Context,
	tenantID string,
	uuid string,
) (clientID, clientSecret string, err error) {
	args := a.Called(ctx, tenantID, uuid)
	return args.String(0), args.String(1), args.Error(errIndex)
}

func (a *authServiceMock) GetCredentialsByUUID(
	ctx context.Context,
	tenantID,
	uuid string,
) (clientID, clientSecret string, err error) {
	args := a.Called(ctx, tenantID, uuid)
	return args.String(0), args.String(1), args.Error(errIndex)
}

func (a *authServiceMock) RevokeCredentialsByUUID(ctx context.Context, tenantID, uuid string) error {
	args := a.Called(ctx, tenantID, uuid)
	return args.Error(0)
}

func (a *authServiceMock) Logout(_ context.Context) {
	a.Called()
}

func AuthServiceMockFactory(createShouldFail, getShouldFail,
	revokeShouldFail bool,
) func(ctx context.Context) (auth.AuthService, error) {
	authMock := &authServiceMock{}

	if createShouldFail {
		authMock.On("CreateCredentialsWithUUID", mock.Anything, mock.Anything, mock.Anything).
			Return("", "", inv_errors.Errorf(""))
	} else {
		authMock.On("CreateCredentialsWithUUID", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}

	if getShouldFail {
		authMock.On("GetCredentialsByUUID", mock.Anything, mock.Anything, mock.Anything).Return("", "", inv_errors.Errorf(""))
	} else {
		authMock.On("GetCredentialsByUUID", mock.Anything, mock.Anything, mock.Anything).Return("", "", nil)
	}

	if revokeShouldFail {
		authMock.On("RevokeCredentialsByUUID", mock.Anything, mock.Anything, mock.Anything).Return(inv_errors.Errorf(""))
	} else {
		authMock.On("RevokeCredentialsByUUID", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	}

	authMock.On("Logout", mock.Anything, mock.Anything).Return()

	return func(_ context.Context) (auth.AuthService, error) {
		return authMock, nil
	}
}
