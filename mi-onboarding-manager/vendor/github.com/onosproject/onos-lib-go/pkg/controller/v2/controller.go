// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"hash/fnv"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger()

const (
	defaultTimeout    = 30 * time.Second
	defaultBufferSize = 100
)

// Options is options for the Controller
type Options struct {
	Log         logging.Logger
	Parallelism int
	BufferSize  int
	Timeout     *time.Duration
}

// Option is a Controller option
type Option func(*Options)

// WithOptions sets the Controller options
func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

// WithLog sets the Controller Logger
func WithLog(log logging.Logger) Option {
	return func(options *Options) {
		options.Log = log
	}
}

// WithParallelism sets the number of parallel goroutines to use for reconciliation
func WithParallelism(parallelism int) Option {
	return func(options *Options) {
		options.Parallelism = parallelism
	}
}

// WithBufferSize sets the buffer size for reconciliation queues
func WithBufferSize(bufferSize int) Option {
	return func(options *Options) {
		options.BufferSize = bufferSize
	}
}

// WithTimeout sets the timeout for each reconciliation cycle
func WithTimeout(timeout time.Duration) Option {
	return func(options *Options) {
		options.Timeout = &timeout
	}
}

// NewController creates a new controller
func NewController[I ID](reconciler Reconciler[I], opts ...Option) *Controller[I] {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}
	bufferSize := options.BufferSize
	if bufferSize == 0 {
		bufferSize = defaultBufferSize
	}
	numPartitions := options.Parallelism
	if numPartitions == 0 {
		numPartitions = 1
	}
	partitions := make([]chan Request[I], numPartitions)
	for i := 0; i < numPartitions; i++ {
		partitions[i] = make(chan Request[I], bufferSize)
	}
	rlog := options.Log
	if rlog == nil {
		rlog = log
	}
	timeout := defaultTimeout
	if options.Timeout != nil {
		timeout = *options.Timeout
	}
	controller := &Controller[I]{
		partitions: partitions,
		reconciler: reconciler,
		Log:        rlog,
		timeout:    timeout,
	}
	controller.start()
	return controller
}

// Controller manages a set of control loops for reconciling objects.
// The type parameter is the type of object identifier reconciled by this controller. To reconcile an object,
// enqueue the object ID by calling the Reconcile method. Once called, the controller will call the configured
// Reconciler to reconcile the request until complete.
type Controller[I ID] struct {
	reconciler Reconciler[I]
	partitions []chan Request[I]
	Log        logging.Logger
	timeout    time.Duration
}

// start starts the controller
func (c *Controller[I]) start() {
	for _, partition := range c.partitions {
		go c.processRequests(partition)
	}
}

// Stop stops the controller
func (c *Controller[I]) Stop() {
	for _, partition := range c.partitions {
		close(partition)
	}
}

// Reconcile reconciles the given object ID
func (c *Controller[I]) Reconcile(id I) error {
	hash, err := computeHash(id)
	if err != nil {
		return err
	}
	request := Request[I]{
		ID:        id,
		partition: hash % len(c.partitions),
	}
	go c.enqueue(request)
	return nil
}

func (c *Controller[I]) enqueue(request Request[I]) {
	request.attempt++
	partition := c.partitions[request.partition]
	partition <- request
}

// processRequests reconciles requests from the given channel
func (c *Controller[I]) processRequests(ch chan Request[I]) {
	for request := range ch {
		c.processRequest(request)
	}
}

// processRequest reconciles the given request
func (c *Controller[I]) processRequest(request Request[I]) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	directive := c.reconciler(ctx, request)
	if directive != nil {
		directive.Do(c)
	}
}

func computeHash[I ID](id I) (int, error) {
	hash := fnv.New32a()
	if _, err := hash.Write([]byte(id.String())); err != nil {
		return 0, err
	}
	return int(hash.Sum32()), nil
}
