// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package reconcilers

import (
	"time"

	rec_v2 "github.com/onosproject/onos-lib-go/pkg/controller/v2"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"

	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
)

const (
	minDelay = 1 * time.Second
	maxDelay = 30 * time.Second
)

type ResourceID string

func (id ResourceID) String() string {
	return string(id)
}

func HandleInventoryError(err error, request rec_v2.Request[ResourceID]) rec_v2.Directive[ResourceID] {
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

func HandleProvisioningError(err error, request rec_v2.Request[ResourceID]) rec_v2.Directive[ResourceID] {
	if _, ok := grpc_status.FromError(err); !ok {
		return request.Ack()
	}

	if inv_errors.IsOperationInProgress(err) {
		// in progress, schedule next reconciliation cycle
		// TODO: it should be Requeue when we remove periodic reconciliation in future
		return request.Retry(err).With(rec_v2.ExponentialBackoff(minDelay, maxDelay))
	}

	if grpc_status.Convert(err).Code() == codes.Aborted {
		// unrecoverable error
		return request.Fail(err)
	}

	if err != nil {
		return request.Retry(err).With(rec_v2.ExponentialBackoff(minDelay, maxDelay))
	}

	return nil
}
