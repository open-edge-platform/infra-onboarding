// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package errors

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/internal/ent"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	location_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/location/v1"
	network_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/network/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	ou_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/ou/v1"
	provider_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/provider/v1"
	schedule_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/schedule/v1"
	tenant_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/tenant/v1"
)

// Mapping codes to HTTP statuses.
var errorCodesToStatus = map[codes.Code]int{
	codes.Canceled:           http.StatusNotAcceptable,
	codes.NotFound:           http.StatusNotFound,
	codes.InvalidArgument:    http.StatusUnprocessableEntity,
	codes.DeadlineExceeded:   http.StatusRequestTimeout,
	codes.AlreadyExists:      http.StatusConflict,
	codes.ResourceExhausted:  http.StatusTooManyRequests,
	codes.FailedPrecondition: http.StatusPreconditionFailed,
	codes.OutOfRange:         http.StatusUnprocessableEntity,
	codes.Unimplemented:      http.StatusNotImplemented,
	codes.Unavailable:        http.StatusServiceUnavailable,
	codes.Unauthenticated:    http.StatusUnauthorized,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.Unknown:            http.StatusInternalServerError,
	codes.Aborted:            http.StatusInternalServerError,
	codes.Internal:           http.StatusInternalServerError,
	codes.DataLoss:           http.StatusInternalServerError,
	codes.OK:                 http.StatusOK,
}

// Mapping reasons to gRPC codes.
var reasonToCode = map[Reason]codes.Code{
	Reason_UNKNOWN_CLIENT: codes.PermissionDenied,
}

// Unhandled codes.
const (
	notACode   = "Code("
	NotAnError = "NOT AN ERROR"
	entRegex   = "(?i)ent: "
	pqRegex    = "(?i)pq: "
)

// build creates a new LP-I error wrapping a reason, and a
// stacktrace into a gRPC status which is converted ultimately
// into an error.
func build(reason Reason, err error) error {
	code, ok := reasonToCode[reason]
	if !ok {
		code = codes.Code(reason)
		if strings.Contains(code.String(), notACode) {
			code = codes.Unknown
		}
	}
	st := grpc_status.New(code, err.Error())
	ei := &ErrorInfo{
		Reason:     reason,
		Stacktrace: fmt.Sprintf("%+v", err),
	}
	st, err = st.WithDetails(ei)
	if err != nil {
		// If this errored, it will always error
		// here, so better panic so we can figure
		// out why than have this silently passing.
		panic(fmt.Sprintf("Unexpected error attaching details: %v", err))
	}
	return st.Err()
}

func sanitizeError(err error) string {
	// Sanitize errors by masking ent
	r := regexp.MustCompile(entRegex)
	errStr := r.ReplaceAllString(err.Error(), "")
	// Sanitize errors by masking pq
	r = regexp.MustCompile(pqRegex)
	return r.ReplaceAllString(errStr, "")
}

// Wrap wraps the error by adding context details and by carry
// it over a grpc status that can be used to log details of
// errors or to print a generic error to the extern.
//
// err is the error to be wrapped.
func Wrap(err error) error {
	if err != nil {
		errStr := sanitizeError(err)
		// Parse ent errors providing a generic mapping
		switch {
		case ent.IsValidationError(err):
			return build(Reason(codes.InvalidArgument), errors.Errorf("%s", errStr))
		case ent.IsConstraintError(err):
			return build(Reason(codes.FailedPrecondition), errors.Errorf("%s", errStr))
		case ent.IsNotFound(err):
			return build(Reason(codes.NotFound), errors.Errorf("%s", errStr))
		case ent.IsNotSingular(err):
			return build(Reason(codes.Internal), errors.Errorf("%s", errStr))
		case ent.IsNotLoaded(err):
			return build(Reason(codes.Internal), errors.Errorf("%s", errStr))
		}
		// Parse validate errors.
		switch {
		case errors.As(err, &inv_v1.UpdateResourceRequestMultiError{}),
			errors.As(err, &inv_v1.UpdateResourceRequestValidationError{}),
			errors.As(err, &computev1.HostResourceMultiError{}),
			errors.As(err, &computev1.HostResourceValidationError{}),
			errors.As(err, &computev1.HoststorageResourceMultiError{}),
			errors.As(err, &computev1.HoststorageResourceValidationError{}),
			errors.As(err, &computev1.HostnicResourceMultiError{}),
			errors.As(err, &computev1.HostnicResourceValidationError{}),
			errors.As(err, &computev1.HostusbResourceMultiError{}),
			errors.As(err, &computev1.HostusbResourceValidationError{}),
			errors.As(err, &location_v1.RegionResourceMultiError{}),
			errors.As(err, &location_v1.RegionResourceValidationError{}),
			errors.As(err, &location_v1.SiteResourceMultiError{}),
			errors.As(err, &location_v1.SiteResourceValidationError{}),
			errors.As(err, &network_v1.EndpointResourceMultiError{}),
			errors.As(err, &network_v1.EndpointResourceValidationError{}),
			errors.As(err, &network_v1.NetlinkResourceMultiError{}),
			errors.As(err, &network_v1.NetlinkResourceValidationError{}),
			errors.As(err, &network_v1.NetworkSegmentMultiError{}),
			errors.As(err, &network_v1.NetworkSegmentValidationError{}),
			errors.As(err, &osv1.OperatingSystemResourceMultiError{}),
			errors.As(err, &osv1.OperatingSystemResourceValidationError{}),
			errors.As(err, &ou_v1.OuResourceMultiError{}),
			errors.As(err, &ou_v1.OuResourceValidationError{}),
			errors.As(err, &provider_v1.ProviderResourceMultiError{}),
			errors.As(err, &provider_v1.ProviderResourceValidationError{}),
			errors.As(err, &schedule_v1.SingleScheduleResourceMultiError{}),
			errors.As(err, &schedule_v1.SingleScheduleResourceValidationError{}),
			errors.As(err, &schedule_v1.RepeatedScheduleResourceMultiError{}),
			errors.As(err, &schedule_v1.RepeatedScheduleResourceValidationError{}),
			errors.As(err, &tenant_v1.ProjectResourceMultiError{}),
			errors.As(err, &tenant_v1.ProjectResourceValidationError{}),
			errors.As(err, &computev1.InstanceResourceMultiError{}),
			errors.As(err, &computev1.InstanceResourceValidationError{}),
			errors.As(err, &computev1.WorkloadResourceMultiError{}),
			errors.As(err, &computev1.WorkloadResourceValidationError{}),
			errors.As(err, &computev1.WorkloadMemberMultiError{}),
			errors.As(err, &computev1.WorkloadMemberValidationError{}),
			errors.As(err, &inv_v1.ResourceFilterMultiError{}),
			errors.As(err, &inv_v1.ResourceFilterValidationError{}),
			errors.As(err, &inv_v1.FindResourcesRequestMultiError{}),
			errors.As(err, &inv_v1.FindResourcesRequestValidationError{}),
			errors.As(err, &inv_v1.ListResourcesRequestMultiError{}),
			errors.As(err, &inv_v1.ListResourcesRequestValidationError{}):
			return build(Reason(codes.InvalidArgument), errors.Errorf("%s", err.Error()))
		}
		// Check if context was canceled
		if errors.Is(err, context.Canceled) {
			return build(Reason(codes.Canceled), errors.Errorf("%s", err.Error()))
		}
		// Check if it is our error and return as it is
		errorInfo := GetErrorInfo(err)
		if errorInfo != nil {
			return err
		}
		// Check if err is a grpc status error
		status, ok := grpc_status.FromError(err)
		if ok {
			return build(Reason(status.Code()), errors.Errorf("%s", status.Message()))
		}
		// Otherwise build using our internal classification
		return build(Reason(codes.Internal), errors.Errorf("%s", err.Error()))
	}
	return err
}

// Errorfc creates an error wrapping a gRPC status. Code is used
// to initialize the gRPC status. The latter that can be
// used to log details of errors or to print a generic error
// to the extern.
//
// code the gRPC code to be used in the gRPC status.
func Errorfc(code codes.Code, format string, args ...interface{}) error {
	if code == codes.OK {
		return nil
	}
	// Add context
	err := errors.Errorf(format, args...)
	return build(Reason(code), err)
}

// Errorfr creates an error wrapping a gRPC status. Reason
// is translated into gRPC code and included in the error
// as well to provide a more detailed reason due to the caller.
// Status that can be used to log details of errors or to
// print a generic error to the extern.
//
// reason the Reason to be carried over the gRPC status.
func Errorfr(reason Reason, format string, args ...interface{}) error {
	if reason == Reason_OK {
		return nil
	}
	// Add context
	err := errors.Errorf(format, args...)
	return build(reason, err)
}

// Errorf creates an error wrapping a gRPC status. The latter
// can be used to log details of errors or to print a generic
// error to the extern. Note this will default to an internal error.
func Errorf(format string, args ...interface{}) error {
	// Add context
	err := errors.Errorf(format, args...)
	return build(Reason(codes.Internal), err)
}

// GetErrorInfo is an helper used in the tests.
func GetErrorInfo(err error) *ErrorInfo {
	st := grpc_status.Convert(err)
	for _, detail := range st.Details() {
		if t, ok := detail.(*ErrorInfo); ok {
			return t
		}
	}
	return nil
}

// IsUnKnownClient is an helper function to check if the error
// is UNKNOWN_CLIENT which means a new registration is necessary.
func IsUnKnownClient(err error) bool {
	errorInfo := GetErrorInfo(err)
	if errorInfo != nil && errorInfo.Reason == Reason_UNKNOWN_CLIENT {
		return true
	}
	return false
}

// IsNotFound is a helper function to check if the error
// is gRPC NOT_FOUND which means the required resource is not found.
func IsNotFound(err error) bool {
	st := grpc_status.Convert(err)
	if st != nil && st.Code() == codes.NotFound {
		return true
	}
	return false
}

// IsCanceled is a helper function to check if the error
// is gRPC CANCELED which means the operation was canceled.
func IsCanceled(err error) bool {
	st := grpc_status.Convert(err)
	if st != nil && st.Code() == codes.Canceled {
		return true
	}
	return false
}

// IsAlreadyExists is a helper function to check if the error
// is gRPC ALREADY_EXISTS which means the required resource already exists.
func IsAlreadyExists(err error) bool {
	st := grpc_status.Convert(err)
	if st != nil && st.Code() == codes.AlreadyExists {
		return true
	}
	return false
}

// IsPermissionDenied is a helper function to check if the error
// is gRPC PERMISSION_DENIED which means the operation is not allowed.
func IsPermissionDenied(err error) bool {
	st := grpc_status.Convert(err)
	if st != nil && st.Code() == codes.PermissionDenied {
		return true
	}
	return false
}

// IsUnauthenticated is a helper function to check if the error
// is gRPC UNAUTHENTICATED which means the client is not authorized to perform operation.
func IsUnauthenticated(err error) bool {
	st := grpc_status.Convert(err)
	if st != nil && st.Code() == codes.Unauthenticated {
		return true
	}
	return false
}

// Consider for the future
// func Append(to, err error) error {
// }

// ErrorToString converts status into string
// without leaking details to the outside.
func ErrorToString(err error) string {
	// not a status -> Unknown
	st := grpc_status.Convert(err)
	if st == nil || st.Code() == codes.OK {
		return NotAnError
	}
	// cut the details and keep only code and reason
	return st.Message()
}

// ErrorToStringWithDetails combines reason and
// stacktrace into an error mesg that can be
// print for debug purposes.
func ErrorToStringWithDetails(err error) string {
	// not a status -> Unknown
	st := grpc_status.Convert(err)
	if st == nil || st.Code() == codes.OK {
		return NotAnError
	}
	errorInfo := GetErrorInfo(err)
	if errorInfo != nil {
		return fmt.Sprintf("%d\n%s", errorInfo.Reason, errorInfo.Stacktrace)
	}
	return st.Message()
}

// ErrorToStatus converts code into a HTTP status.
func ErrorToHTTPStatus(err error) int {
	// not a status -> Unknown
	st := grpc_status.Convert(err)
	// actual conversion
	code := st.Code()
	errorStatus, ok := errorCodesToStatus[code]
	if !ok {
		errorStatus = http.StatusInternalServerError
	}
	return errorStatus
}
