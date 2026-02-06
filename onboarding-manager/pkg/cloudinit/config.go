// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package cloudinit provides functionality for onboarding management.
package cloudinit

import (
	"google.golang.org/grpc/codes"

	osv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

// Option provides functionality for onboarding management.
type Option func(*cloudInitOptions)

type cloudInitOptions struct {
	// OsType type of OS for which a cloud-init is generated.
	OsType osv1.OsType

	// RunAsStandalone set to skip provisioning runtime EMF configuration,
	// so that ENs will operate as standalone, unmanaged nodes.
	RunAsStandalone bool

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
	// useLocalAccount set to create local account for SSH access
	useLocalAccount bool
	// localAccountUserName a user name to log in to a local account
	localAccountUserName string
	// sshKey contains public SSH key to be set as authorized key for local account
	sshKey string
}

func defaultCloudInitOptions() cloudInitOptions {
	return cloudInitOptions{
		// be explicit
		RunAsStandalone: false,
		useDevMode:      false,
		OsType:          osv1.OsType_OS_TYPE_UNSPECIFIED,
		useLocalAccount: false,
	}
}

//nolint:cyclop // we need complex validation logic
func (opts cloudInitOptions) validate() error {
	if opts.OsType == osv1.OsType_OS_TYPE_UNSPECIFIED {
		return inv_errors.Errorfc(codes.InvalidArgument, "Unsupported OS type: %s", opts.OsType.String())
	}

	if opts.hostname == "" {
		return inv_errors.Errorfc(codes.InvalidArgument, "Hostname must be provided")
	}

	if opts.hostMAC == "" {
		return inv_errors.Errorfc(codes.InvalidArgument, "Host's MAC address must be provided")
	}

	if opts.useDevMode && (opts.devUsername == "" || opts.devUserPasswd == "") {
		return inv_errors.Errorfc(codes.InvalidArgument,
			"Username and password must be provided if dev mode is enabled")
	}

	if opts.useLocalAccount && (opts.localAccountUserName == "" || opts.sshKey == "") {
		return inv_errors.Errorfc(codes.InvalidArgument,
			"Username and SSH key must be provided if local account is enabled is there for an instance")
	}

	if opts.preserveIP && (opts.staticHostIP == "" || len(opts.staticDNS) == 0) {
		return inv_errors.Errorfc(codes.InvalidArgument,
			"IP address to set must be provided if static IP enabled")
	}

	if !opts.RunAsStandalone {
		if err := opts.validateNonStandaloneOptions(); err != nil {
			return err
		}
	}

	return nil
}

func (opts cloudInitOptions) validateNonStandaloneOptions() error {
	if opts.tenantID == "" {
		return inv_errors.Errorfc(codes.InvalidArgument, "Tenant ID must be provided")
	}

	if opts.clientID == "" || opts.clientSecret == "" {
		return inv_errors.Errorfc(codes.InvalidArgument, "Client credentials must be provided")
	}

	return nil
}

// WithRunAsStandalone performs operations for onboarding management.
func WithRunAsStandalone() Option {
	return func(options *cloudInitOptions) {
		options.RunAsStandalone = true
	}
}

// WithDevMode performs operations for onboarding management.
func WithDevMode(username, password string) Option {
	return func(options *cloudInitOptions) {
		options.useDevMode = true
		options.devUsername = username
		options.devUserPasswd = password
	}
}

// WithOSType performs operations for onboarding management.
func WithOSType(osType osv1.OsType) Option {
	return func(options *cloudInitOptions) {
		options.OsType = osType
	}
}

// WithTenantID performs operations for onboarding management.
func WithTenantID(tenantID string) Option {
	return func(options *cloudInitOptions) {
		options.tenantID = tenantID
	}
}

// WithHostname performs operations for onboarding management.
func WithHostname(hostname string) Option {
	return func(options *cloudInitOptions) {
		options.hostname = hostname
	}
}

// WithClientCredentials performs operations for onboarding management.
func WithClientCredentials(clientID, clientSecret string) Option {
	return func(options *cloudInitOptions) {
		options.clientID = clientID
		options.clientSecret = clientSecret
	}
}

// WithLocalAccount performs operations for onboarding management.
func WithLocalAccount(localAccountUserName, sshKey string) Option {
	return func(options *cloudInitOptions) {
		options.useLocalAccount = true
		options.localAccountUserName = localAccountUserName
		options.sshKey = sshKey
	}
}

// WithHostMACAddress performs operations for onboarding management.
func WithHostMACAddress(mac string) Option {
	return func(options *cloudInitOptions) {
		options.hostMAC = mac
	}
}

// WithPreserveIP performs operations for onboarding management.
func WithPreserveIP(hostIP string, dnsServers []string) Option {
	return func(options *cloudInitOptions) {
		options.preserveIP = true
		options.staticHostIP = hostIP
		options.staticDNS = dnsServers
	}
}
