// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"

	inv_errors "github.com/intel/infra-core/inventory/v2/pkg/errors"
	rec_v2 "github.com/intel/orch-library/go/pkg/controller/v2"
)

const (
	minDelay = 1 * time.Second
	maxDelay = 60 * time.Second
)

var (
	retryMinDelay = minDelay
	retryMaxDelay = maxDelay
)

// Inventory resource IDs + tenant IDs are used to feed reconciler functions.
type ReconcilerID string

func WrapReconcilerID(tenantID, resourceID string) ReconcilerID {
	return ReconcilerID(tenantID + "/" + resourceID)
}

func UnwrapReconcilerID(id ReconcilerID) (tenantID, resourceID string) {
	unwrapped := strings.Split(id.String(), "/")
	return unwrapped[0], unwrapped[1]
}

func (id ReconcilerID) String() string {
	return string(id)
}

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
