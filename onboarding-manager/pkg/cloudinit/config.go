// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"google.golang.org/grpc/codes"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

type Option func(*cloudInitOptions)

type cloudInitOptions struct {
	// OsType type of OS for which a cloud-init is generated.
	OsType osv1.OsType

	// useDevMode enables creation of local admin user
	useDevMode bool
	// username defines a username for local admin user, must be provided if useDevMode is set
	devUsername string
	// userPasswd defines a password for local admin user, must be provided if useDevMode is set
	devUserPasswd string

	// tenantID specifies UUID of tenant that Host belongs to
	tenantID string
	// hostname specifies host name to be set on Host
	hostname string

	// clientID specifies client ID used to obtain JWT token for authorization
	clientID string
	// clientSecret specifies client secret used to obtain JWT token for authorization
	clientSecret string
}

func defaultCloudInitOptions() cloudInitOptions {
	return cloudInitOptions{
		// be explicit
		useDevMode: false,
		OsType:     osv1.OsType_OS_TYPE_UNSPECIFIED,
	}
}

func (opts cloudInitOptions) validate() error {
	if opts.OsType == osv1.OsType_OS_TYPE_UNSPECIFIED {
		return inv_errors.Errorfc(codes.InvalidArgument, "Unsupported OS type: %s", opts.OsType.String())
	}

	if opts.tenantID == "" {
		return inv_errors.Errorfc(codes.InvalidArgument, "Tenant ID must be provided")
	}

	if opts.hostname == "" {
		return inv_errors.Errorfc(codes.InvalidArgument, "Hostname must be provided")
	}

	if opts.clientID == "" || opts.clientSecret == "" {
		return inv_errors.Errorfc(codes.InvalidArgument, "Client credentials must be provided")
	}

	if opts.useDevMode && (opts.devUsername == "" || opts.devUserPasswd == "") {
		return inv_errors.Errorfc(codes.InvalidArgument,
			"Username and password must be provided if dev mode is enabled")
	}

	return nil
}

func WithDevMode(username, password string) Option {
	return func(options *cloudInitOptions) {
		options.useDevMode = true
		options.devUsername = username
		options.devUserPasswd = password
	}
}

func WithOSType(osType osv1.OsType) Option {
	return func(options *cloudInitOptions) {
		options.OsType = osType
	}
}

func WithTenantID(tenantID string) Option {
	return func(options *cloudInitOptions) {
		options.tenantID = tenantID
	}
}

func WithHostname(hostname string) Option {
	return func(options *cloudInitOptions) {
		options.hostname = hostname
	}
}

func WithClientCredentials(clientID, clientSecret string) Option {
	return func(options *cloudInitOptions) {
		options.clientID = clientID
		options.clientSecret = clientSecret
	}
}
