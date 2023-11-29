// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"math"
	"time"
)

// ID is an interface to be implemented for object identifiers
type ID interface {
	~string | ~int | ~int32 | ~int64 | ~uint | ~uint32 | ~uint64
	fmt.Stringer
}

// Reconciler reconciles an object
type Reconciler[I ID] func(ctx context.Context, request Request[I]) Directive[I]

// Directive is a controller directive indicating how to proceed after reconciliation
type Directive[I ID] interface {
	Do(controller *Controller[I])
}

// Request is a reconciler request
type Request[I ID] struct {
	ID        I
	partition int
	attempt   int
}

// Ack acknowledges the request was reconciled successfully, removing it from the reconciliation queue.
func (r Request[I]) Ack() *Ack[I] {
	return &Ack[I]{
		request: r,
	}
}

// Requeue acknowledges successful reconciliation of the request, requeueing the request for further reconciliation.
func (r Request[I]) Requeue() *Requeue[I] {
	return &Requeue[I]{
		request: r,
	}
}

// Fail fails reconciliation of the request, logging the given error and removing it from the reconciliation queue.
func (r Request[I]) Fail(err error) *Fail[I] {
	return &Fail[I]{
		request: r,
		Error:   err,
	}
}

// Retry logs a reconciliation error and retries reconciliation of the request.
func (r Request[I]) Retry(err error) *Retry[I] {
	return &Retry[I]{
		request: r,
		Error:   err,
	}
}

// Backoff is a function for computing the backoff duration following a failed request
type Backoff func(attempt int) time.Duration

// ConstantBackoff computes a constant backoff delay
func ConstantBackoff(delay time.Duration, maxDelay time.Duration) Backoff {
	return func(attempt int) time.Duration {
		return time.Duration(math.Min(float64(int(delay)*attempt), float64(maxDelay)))
	}
}

// ExponentialBackoff computes an exponentially increasing backoff delay
func ExponentialBackoff(initialDelay time.Duration, maxDelay time.Duration) Backoff {
	return func(attempt int) time.Duration {
		maxExponent := math.Log2(float64(maxDelay) / float64(initialDelay))
		return initialDelay * time.Duration(math.Pow(2, math.Min(float64(attempt), maxExponent)))
	}
}

// Ack acknowledges a reconciliation request
type Ack[I ID] struct {
	request Request[I]
}

// Do executes the controller directive
func (c *Ack[I]) Do(controller *Controller[I]) {
	controller.Log.Debugw("Reconciliation complete", "Request.ID", c.request.ID)
}

// Requeue requeues a reconciliation request
type Requeue[I ID] struct {
	request Request[I]
}

// Do executes the controller directive
func (r *Requeue[I]) Do(controller *Controller[I]) {
	controller.Log.Debugw("Requeueing request", "Request.ID", r.request.ID)
	go controller.enqueue(r.request)
}

// Fail fails a reconciliation request
type Fail[I ID] struct {
	request Request[I]
	Error   error
}

// Do executes the controller directive
func (f *Fail[I]) Do(controller *Controller[I]) {
	controller.Log.Warnw("Reconciliation failed", "Request.ID", f.request.ID, "Error", f.Error.Error())
}

// Retry retries a reconciliation request
type Retry[I ID] struct {
	request Request[I]
	Error   error
}

// Do executes the controller directive
func (r *Retry[I]) Do(controller *Controller[I]) {
	controller.Log.Debugw("Reconciliation failed. Retrying...", "Request.ID", r.request.ID, "Error", r.Error.Error())
	go controller.enqueue(r.request)
}

// After retries the request after the given delay
func (r *Retry[I]) After(delay time.Duration) *RetryAfter[I] {
	return &RetryAfter[I]{
		Retry: r,
		delay: delay,
	}
}

// At retries the request at the given time
func (r *Retry[I]) At(t time.Time) *RetryAt[I] {
	return &RetryAt[I]{
		Retry: r,
		t:     t,
	}
}

// With retries the request with the given backoff policy
func (r *Retry[I]) With(backoff Backoff) *RetryWith[I] {
	return &RetryWith[I]{
		Retry:   r,
		backoff: backoff,
	}
}

// RetryAfter retries a reconciliation request after a delay
type RetryAfter[I ID] struct {
	*Retry[I]
	delay time.Duration
}

// Do executes the controller directive
func (r *RetryAfter[I]) Do(controller *Controller[I]) {
	controller.Log.Debugw("Reconciliation failed. Retrying...", "Request.ID", r.request.ID, "Error", r.Error.Error(), "Delay", r.delay)
	time.AfterFunc(r.delay, func() {
		controller.enqueue(r.request)
	})
}

// RetryAt retries a reconciliation request at a specific time
type RetryAt[I ID] struct {
	*Retry[I]
	t time.Time
}

// Do executes the controller directive
func (r *RetryAt[I]) Do(controller *Controller[I]) {
	controller.Log.Debugw("Reconciliation failed. Retrying...", "Request.ID", r.request.ID, "Error", r.Error.Error(), "Time", r.t)
	time.AfterFunc(time.Until(r.t), func() {
		controller.enqueue(r.request)
	})
}

// RetryWith retries a reconciliation request with a backoff policy
type RetryWith[I ID] struct {
	*Retry[I]
	backoff Backoff
}

// Do executes the controller directive
func (r *RetryWith[I]) Do(controller *Controller[I]) {
	delay := r.backoff(r.request.attempt)
	controller.Log.Debugw("Reconciliation failed. Retrying...", "Request.ID", r.request.ID, "Error", r.Error.Error(), "Delay", delay)
	time.AfterFunc(delay, func() {
		controller.enqueue(r.request)
	})
}
