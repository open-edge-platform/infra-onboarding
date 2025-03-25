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

	// hostMAC sets MAC address of host's management interface to be provided to netplan to speed up network discovery process
	hostMAC string
	// preserveIP if enabled, preserves IP address that is initially auto-assigned by DHCP during uOS stage
	preserveIP bool
	// staticHostIP static IP address to configure on a Host
	staticHostIP string
	// staticDNS set of statically configured DNS servers, must be provided if preserveIP is true
	staticDNS []string
}

func defaultCloudInitOptions() cloudInitOptions {
	return cloudInitOptions{
		// be explicit
		useDevMode: false,
		OsType:     osv1.OsType_OS_TYPE_UNSPECIFIED,
	}
}

//nolint:cyclop // we need complex validation logic
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

	if opts.hostMAC == "" {
		return inv_errors.Errorfc(codes.InvalidArgument, "Host's MAC address must be provided")
	}

	if opts.clientID == "" || opts.clientSecret == "" {
		return inv_errors.Errorfc(codes.InvalidArgument, "Client credentials must be provided")
	}

	if opts.useDevMode && (opts.devUsername == "" || opts.devUserPasswd == "") {
		return inv_errors.Errorfc(codes.InvalidArgument,
			"Username and password must be provided if dev mode is enabled")
	}

	if opts.preserveIP && (opts.staticHostIP == "" || len(opts.staticDNS) == 0) {
		return inv_errors.Errorfc(codes.InvalidArgument,
			"IP address to set must be provided if static IP enabled")
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

func WithHostMACAddress(mac string) Option {
	return func(options *cloudInitOptions) {
		options.hostMAC = mac
	}
}

func WithPreserveIP(hostIP string, dnsServers []string) Option {
	return func(options *cloudInitOptions) {
		options.preserveIP = true
		options.staticHostIP = hostIP
		options.staticDNS = dnsServers
	}
}
