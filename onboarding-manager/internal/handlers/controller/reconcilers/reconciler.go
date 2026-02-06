// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	rec_v2 "github.com/open-edge-platform/orch-library/go/pkg/controller/v2"
)

const (
	minDelay = 1 * time.Second
	maxDelay = 60 * time.Second
)

var (
	retryMinDelay = minDelay
	retryMaxDelay = maxDelay
)

// ReconcilerID provides functionality for onboarding management.
// Inventory resource IDs + tenant IDs are used to feed reconciler functions.
type ReconcilerID string

func (id ReconcilerID) String() string {
	return fmt.Sprintf("[tenantID=%s, resourceID=%s]", id.GetTenantID(), id.GetResourceID())
}

// GetTenantID performs operations for the receiver.
func (id ReconcilerID) GetTenantID() string {
	return strings.Split(string(id), "_")[0]
}

// GetResourceID performs operations for the receiver.
func (id ReconcilerID) GetResourceID() string {
	return strings.Split(string(id), "_")[1]
}

// NewReconcilerID performs operations for onboarding management.
func NewReconcilerID(tenantID, resourceID string) ReconcilerID {
	return ReconcilerID(fmt.Sprintf("%s_%s", tenantID, resourceID))
}

// HandleInventoryError performs operations for onboarding management.
func HandleInventoryError(err error, request rec_v2.Request[ReconcilerID]) rec_v2.Directive[ReconcilerID] {
	if _, ok := grpc_status.FromError(err); !ok {
		return request.Ack()
	}

	if inv_errors.IsNotFound(err) || inv_errors.IsAlreadyExists(err) ||
		inv_errors.IsUnauthenticated(err) || inv_errors.IsPermissionDenied(err) {
		return request.Ack()
	}

	if err != nil {
		return request.Retry(err).With(rec_v2.ExponentialBackoff(minDelay, maxDelay))
	}

	return nil
}

// HandleProvisioningError performs operations for onboarding management.
func HandleProvisioningError(err error, request rec_v2.Request[ReconcilerID]) rec_v2.Directive[ReconcilerID] {
	if _, ok := grpc_status.FromError(err); !ok {
		return request.Ack()
	}

	if inv_errors.IsOperationInProgress(err) {
		// in progress, schedule next reconciliation cycle
		// TODO: it should be Requeue when we remove periodic reconciliation in future
		return request.Retry(err).With(rec_v2.ExponentialBackoff(retryMinDelay, retryMaxDelay))
	}

	if grpc_status.Convert(err).Code() == codes.Aborted {
		// unrecoverable error
		return request.Fail(err)
	}

	if err != nil {
		return request.Retry(err).With(rec_v2.ExponentialBackoff(retryMinDelay, retryMaxDelay))
	}

	return nil
}
