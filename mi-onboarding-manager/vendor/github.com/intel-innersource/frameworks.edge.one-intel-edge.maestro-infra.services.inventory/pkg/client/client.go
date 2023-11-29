// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package client

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	inv_v1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/inventory/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/cert"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/tracing"
)

var zlog = logging.GetLogger("MIAPIClient")

const BatchSize = 50

type WatchEvents struct {
	Ctx   context.Context
	Event *inv_v1.SubscribeEventsResponse
}

type inventoryClient struct {
	cfg          *InventoryClientConfig
	connection   *grpc.ClientConn
	invAPI       inv_v1.InventoryServiceClient
	clientUUID   string
	streamCtx    context.Context
	streamCancel context.CancelFunc
	stream       inv_v1.InventoryService_SubscribeEventsClient
	uuidMutex    sync.RWMutex
}

// InventoryClient defines all the methods that inventoryClient
// must implement.
type InventoryClient interface {
	// Close unregisters the client from the inventory server and terminates the
	// gRPC connection.
	Close() error
	// List looks for inventory resources based on a filter definition
	// returning their objects.
	List(context.Context, *inv_v1.ResourceFilter) (*inv_v1.ListResourcesResponse, error)
	// ListAll looks for inventory resources based on the given filter and fieldMask
	// returning all objects that matches the filter.
	ListAll(context.Context, *inv_v1.Resource, *fieldmaskpb.FieldMask) ([]*inv_v1.Resource, error)
	// Find looks for inventory resources based on a filter definition
	// returning their IDs.
	Find(context.Context, *inv_v1.ResourceFilter) (*inv_v1.FindResourcesResponse, error)
	// FindAll looks for inventory resources based on the given filter and fieldMask
	// returning all the ID that matches the filter
	FindAll(context.Context, *inv_v1.Resource, *fieldmaskpb.FieldMask) ([]string, error)
	// Get retrieves a resource from inventory based on its ID.
	Get(context.Context, string) (*inv_v1.GetResourceResponse, error)
	// Create creates a resource in inventory, providing its newly created ID in the response.
	Create(context.Context, *inv_v1.Resource) (*inv_v1.CreateResourceResponse, error)
	// Update updates a resource in inventory, given the resource ID, the fieldmask
	// to be applied on the resource fields, and the resource instance.
	Update(context.Context, string, *fieldmaskpb.FieldMask, *inv_v1.Resource) (*inv_v1.UpdateResourceResponse, error)
	// Delete deletes a resource from inventory based on its ID.
	Delete(context.Context, string) (*inv_v1.DeleteResourceResponse, error)
	// TestingOnlySetClient allows to set the internal inventory service client
	// API for testing purposes only.
	TestingOnlySetClient(inv_v1.InventoryServiceClient)
}

// isRetryableStreamError checks if a registration error is recoverable and a
// new register retry should be performed.
func isRetryableStreamError(err error) bool {
	if errors.Is(err, io.EOF) {
		zlog.MiSec().MiErr(err).Msg("Inventory client stream gracefully disconnected")
		return true
	}

	if code := status.Code(err); code == codes.Unavailable {
		zlog.MiSec().MiErr(err).Msg("Inventory client stream unavailable")
		return true
	}

	return false
}

// registerRetryBackoffLoop runs a loop to retry register the inventory client.
// It returns an error in case the max elapsed time of expbackoff was attained,
// the inventory client was terminated, or if the stream error does not allow to
// retry the register.
func (client *inventoryClient) registerRetryBackoffLoop(expbackoff *backoff.ExponentialBackOff) error {
	for {
		// Try to register again
		zlog.MiSec().Debug().Msgf("Client register retry, elapsed time %v", expbackoff.GetElapsedTime())
		err := client.register()

		// If register ok, break and return nil
		if err == nil {
			zlog.MiSec().Info().Msg("Client register retry successful")
			return nil
		}

		// Checks if register error is retryable
		if !isRetryableStreamError(err) {
			zlog.MiSec().MiErr(err).Msg("Register retry loop finished, stream error not retryable")
			return err
		}

		// Gets next backoff time and checks if it is still valid
		d := expbackoff.NextBackOff()
		if d == backoff.Stop {
			err := inv_errors.Errorfc(codes.Internal, "backoff max elapsed time")
			zlog.MiSec().MiErr(err).Msg("Register retry loop finished")
			return err
		}

		select {
		// Waits/sleeps during backoff time
		case <-time.After(d):
			zlog.MiSec().Debug().Msgf("Client waited on next register retry for %v", d)

		// Verifies if TermChan is enabled to stop backoff inner loop.
		case <-client.cfg.TermChan:
			err := inv_errors.Errorfc(codes.Internal, "inventory client terminated")
			zlog.MiSec().MiErr(err).Msg("Register retry loop finished")
			return err

		// Waits for client context to be done
		case <-client.streamCtx.Done():
			err := inv_errors.Errorfc(codes.Internal, "finished register retry loop")
			zlog.MiSec().MiErr(err).Msg("inventory client stream context done")
			return err
		}
	}
}

// registerRetry is a helper method to be used to perform registration
// retries when the client stream context is closed.
// Once the subscription to events was interrupted or finished.
// It uses an exponential backoff timer to wait between retries.
// Its backoff mechanism is configured with the RegisterMaxElapsedTime of InventoryClientConfig.
func (client *inventoryClient) registerRetry() error {
	// Checks if register retry is enabled, otherwise returns error.
	if !client.cfg.EnableRegisterRetry {
		err := inv_errors.Errorfc(codes.Internal, "regiter retry not enabled")
		zlog.MiSec().MiErr(err).Msg("could not retry register")
		return err
	}

	expbackoff := backoff.NewExponentialBackOff()
	expbackoff.MaxElapsedTime = client.cfg.RegisterMaxElapsedTime

	err := client.registerRetryBackoffLoop(expbackoff)
	if err != nil {
		return err
	}

	return nil
}

// streamClosedHandler is a helper function of inventory client.
// It invalidates the clientUUID once the subscription stream is closed.
// I.e., no client should make calls without a valid UUID.
func (client *inventoryClient) streamClosedHandler() {
	select {
	case <-client.stream.Context().Done():
		client.uuidMutex.Lock()
		client.clientUUID = ""
		client.uuidMutex.Unlock()
		zlog.MiSec().Info().Msg("Inventory client stream disconnected, client unregistered")
	default:
		return
	}
}

func (client *inventoryClient) handleStreamErr(err error) error {
	zlog.MiSec().Info().Msg("Handling Inventory client stream error")

	// Invalidate client UUID
	client.streamClosedHandler()

	// server canceled the stream, return error to end event loop
	if inv_errors.IsCanceled(err) {
		zlog.MiSec().Info().Msg("Inventory client stream canceled")
		return err
	}

	select {
	// If stream context is done, go for retry (if enabled).
	case <-client.stream.Context().Done():
		err = client.registerRetry()

		// registerRetry returns error if retry fails or if retry is not enabled
		if err != nil {
			zlog.MiSec().MiErr(err).Msg("Inventory client disconnected")
			return err
		}
		// registerRetry went well, no error to report
		return nil

	// By default, if error happened and stream context is not done,
	// return it to end event loop
	default:
		zlog.MiSec().MiErr(err).Msg("Inventory client disconnected")
		return err
	}
}

// eventContextTracing adds trace from stream header metadata to the context
// when tracing is enabled.
func (client *inventoryClient) eventContextTracing() context.Context {
	ctx := client.stream.Context()
	if client.cfg.EnableTracing {
		// Gets tracing info from stream header into metadata
		md, err := client.stream.Header()
		if err != nil {
			zlog.MiErr(err).Msgf("could not read stream header metadata")
		}
		// Creates new context with tracing info from metadata
		ctx = metadata.NewIncomingContext(client.stream.Context(), md)
		// Sets a new span to the watch
		ctx = tracing.StartTraceFromRemote(ctx, client.cfg.Name, "watch")
		tracing.StopTrace(ctx)
	}
	return ctx
}

// eventHandler will listen for inventory events and enqueue them internal
// channel, which can be accessed with EventChannel. This function blocks until a
// signal over termChan is sent or the server closes the connection. It is safe
// to have a goroutine call this function and another goroutine calling Find,
// Get, Create or Update at the same time, but it is not safe to call eventHandler
// in different goroutines.
func (client *inventoryClient) eventHandler() {
	client.cfg.Wg.Add(1)
	defer client.cfg.Wg.Done()
	defer close(client.cfg.Events) // Only the sender can safely close a channel.

	for {
		// Wait for next event.
		event, err := client.stream.Recv()
		// Checks stream error for retry register (if enabled).
		if err != nil {
			streamErr := client.handleStreamErr(err)
			if streamErr != nil {
				// Cannot retry register or failed doing it, need to stop event
				// loop handler.
				zlog.MiSec().Info().Msg("event stream handler loop finished")
				return
			}
			// Tried register retry and succeeded, need to jump to next loop,
			// because event is nil.
			continue
		}
		// Adds tracing, if enabled, to the event context.
		ctx := client.eventContextTracing()

		// Put event in queue or drop. Non-blocking.
		select {
		case client.cfg.Events <- &WatchEvents{ctx, event}:
		default:
		}
	}
}

func (client *inventoryClient) Close() error {
	client.streamCancel()
	err := inv_errors.Wrap(client.connection.Close())
	return err
}

// termChanHandler waits for a termination signal over termChan.
// If true is sent over the channel, the client will initiate stream channel
// shutdown.
func (client *inventoryClient) termChanHandler() {
	client.cfg.Wg.Add(1)
	defer client.cfg.Wg.Done()
	termSig := <-client.cfg.TermChan
	if termSig {
		err := client.Close()
		zlog.Info().Err(err).Msg("stopping inventory client")
	}
}

// connect creates a gRPC connection to a server.
func connect(
	ctx context.Context,
	address string,
	caPath, certPath, keyPath string,
	insec bool,
	opts ...grpc.DialOption,
) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn

	if insec {
		dialOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
		opts = append(opts, dialOpt)
	} else {
		if caPath == "" || certPath == "" || keyPath == "" {
			err := inv_errors.Errorf("CaCertPath %s or TlsCerPath %s or TlsKeyPath %s were not provided",
				caPath, certPath, keyPath,
			)
			zlog.Fatal().Err(err).Msgf("CaCertPath %s or TlsCerPath %s or TlsKeyPath %s were not provided\n",
				caPath, certPath, keyPath,
			)
			return nil, err
		}
		// setting secure gRPC connection
		creds, err := cert.HandleCertPaths(caPath, keyPath, certPath, true)
		if err != nil {
			zlog.Fatal().Err(err).Msgf("an error occurred while loading credentials to server %v, %v, %v: %v\n",
				caPath, certPath, keyPath, err,
			)
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	// if testing, use a bufconn, otherwise TCP
	var err error
	if address == "bufconn" {
		conn, err = grpc.DialContext(ctx, "", opts...)
	} else {
		conn, err = grpc.DialContext(ctx, address, opts...)
	}
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Unable to dial connection to inventory client address %s", address)
		return nil, inv_errors.Wrap(err)
	}
	return conn, nil
}

// SecurityConfig security configuration for inventoryClient.
// CaPath, CertPath, KeyPath to be set if needed.
// Insecure determines whether to use TLS and requires the above fields to be set.
type SecurityConfig struct {
	CaPath   string
	KeyPath  string
	CertPath string
	Insecure bool
}

// InventoryClientConfig comprises a set of inventory client configuration options.
type InventoryClientConfig struct {
	// Name allows registering this client with an unique name.
	Name string
	// Address is the inventory target to connect to.
	Address string
	// Dial options of this client. This might include also the interceptors
	DialOptions []grpc.DialOption
	// Events define the channel to receive subscription events from inventory.
	// Each event is received together with its incoming context, to be
	// used for tracing purposes.
	Events chan *WatchEvents
	// EnableRegisterRetry determines if a control loop to try to register in case the
	// subscription stream is closed.
	// To avoid race conditions, EnableRegisterRetry and AbortOnUnknownClientError should never be enabled together.
	EnableRegisterRetry bool
	// AbortOnUnknownClientError determines if the inventory client should abort on
	// UNKNOWN_CLIENT error received from Inventory. If AbortOnUnknownClientError is enabled,
	// the inventory client will fatal on UNKNOWN_CLIENT error received, causing a crash of the client's user.
	// To avoid race conditions, EnableRegisterRetry and AbortOnUnknownClientError should never be enabled together.
	AbortOnUnknownClientError bool
	// RegisterMaxElapsedTime is the max time allowed to retry registration procedures
	// in case RegisterRetry is enabled, if set to zero allows registration retries to run indefinitely.
	RegisterMaxElapsedTime time.Duration
	// ClientKind should be set to the appropriate enum value, depending on the type of application.
	ClientKind inv_v1.ClientKind
	// ResourceKinds is a list of resource kinds this client would like to receive
	// updates for.
	ResourceKinds []inv_v1.ResourceKind
	// EnableTracing enables tracing.
	EnableTracing bool
	// TermChan is used to terminate the client.
	TermChan chan bool
	// Wg will be unblocked upon termination of client.
	Wg *sync.WaitGroup
	// SecurityConfig security configuration for inventoryClient.
	SecurityCfg *SecurityConfig
}

// Return an error if user does not provide required input.
func validateClientInput(ctx context.Context, cfg InventoryClientConfig) error {
	if ctx == nil {
		zlog.MiSec().MiError("context is nil").Msg("")
		return inv_errors.Errorfc(codes.InvalidArgument, "context is nil")
	}
	if cfg.TermChan == nil {
		zlog.MiSec().MiError("termChan is nil").Msg("")
		return inv_errors.Errorfc(codes.InvalidArgument, "termChan is nil")
	}
	if cfg.Wg == nil {
		zlog.MiSec().MiError("waitgroup is nil").Msg("")
		return inv_errors.Errorfc(codes.InvalidArgument, "waitgroup is nil")
	}
	if cfg.Events == nil {
		zlog.MiSec().MiError("events channel is nil").Msg("")
		return inv_errors.Errorfc(codes.InvalidArgument, "events channel is nil")
	}
	if cfg.EnableRegisterRetry && cfg.AbortOnUnknownClientError {
		strErr := "Both EnableRegisterRetry and AbortOnUnknownClientError cannot be enabled."
		zlog.MiSec().MiError(strErr).Msg("")
		return inv_errors.Errorfc(codes.InvalidArgument, strErr)
	}
	return nil
}

// NewInventoryClient creates a new inventoryClient with a connection to inventory.
// ctx allows passing in a custom context.Context.
// Users should call inventoryClient.Close to terminate the gRPC connection after this function returns.
func NewInventoryClient(
	ctx context.Context,
	cfg InventoryClientConfig,
) (InventoryClient, error) {
	// Handle required input
	if err := validateClientInput(ctx, cfg); err != nil {
		return nil, err
	}
	// User might not provide dial options
	if cfg.DialOptions == nil {
		cfg.DialOptions = make([]grpc.DialOption, 0)
	}
	if cfg.EnableTracing {
		cfg.DialOptions = append(cfg.DialOptions, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	}

	// ToDo remove insec option as default connect mode
	conn, err := connect(ctx,
		cfg.Address,
		cfg.SecurityCfg.CaPath,
		cfg.SecurityCfg.CertPath,
		cfg.SecurityCfg.KeyPath,
		cfg.SecurityCfg.Insecure,
		cfg.DialOptions...)
	if err != nil {
		return nil, err
	}

	invClient := inv_v1.NewInventoryServiceClient(conn)
	zlog.Debug().Msgf("Created inventory client to address: %s", cfg.Address)

	cl := &inventoryClient{
		cfg:        &cfg,
		connection: conn,
		invAPI:     invClient,
	}

	// registering client and obtaining UUID
	err = cl.register()
	if err != nil {
		// stream is already cancel. Close the connection only
		cl.connection.Close()
		return nil, err
	}

	// Setup handler for user initiated shutdown.
	go cl.termChanHandler()

	// Setup inventory event handler, register retry inside of it.
	go cl.eventHandler()

	return cl, nil
}

// register registers the inventory client on a name and a list of resource kinds.
// It is meant to be used by any register retry go routine that can be called
// once the subscriptions stream context is closed by any unexpected reasons.
// Look at RegisterRetry method for a helper example.
func (client *inventoryClient) register() error {
	ctx, cancel := context.WithCancel(context.Background())
	zlog := zlog.TraceCtx(ctx)
	zlog.MiSec().Info().Msgf("Register inventory client: name %s, clientkind %v, prefixes %s",
		client.cfg.Name, client.cfg.ClientKind, client.cfg.ResourceKinds,
	)

	// Register client by setting up the stream channel.
	req := inv_v1.SubscribeEventsRequest{
		Name:                    client.cfg.Name,
		Version:                 "0.1.0-dev", // TODO: pull version main.Version
		ClientKind:              client.cfg.ClientKind,
		SubscribedResourceKinds: client.cfg.ResourceKinds,
	}
	stream, err := client.invAPI.SubscribeEvents(ctx, &req)
	if err != nil {
		cancel()
		return inv_errors.Wrap(err)
	}
	// Get our UUID from the first response.
	resp, err := stream.Recv()
	if err != nil {
		cancel()
		zlog.MiSec().MiErr(err).Msg("Unable to register inventory client")
		return inv_errors.Wrap(err)
	}
	if resp.ClientUuid == "" {
		cancel()
		zlog.MiError("Server did not allocate an UUID unable to register inventory client").Msg("")
		return inv_errors.Errorfc(codes.Internal, "Server did not allocate an UUID unable to register inventory client")
	}
	// let's close the send half of the stream as we don't need it
	if err := stream.CloseSend(); err != nil {
		zlog.Warn().Msg("unable to close send")
	}
	client.uuidMutex.Lock()
	client.stream = stream
	client.streamCtx = ctx
	client.streamCancel = cancel
	client.clientUUID = resp.ClientUuid
	zlog.MiSec().Info().Msgf("Registered inventory client with UUID: %s", resp.ClientUuid)
	client.uuidMutex.Unlock()

	return nil
}

func (client *inventoryClient) List(
	ctx context.Context,
	filter *inv_v1.ResourceFilter,
) (*inv_v1.ListResourcesResponse, error) {
	zlog := zlog.TraceCtx(ctx)
	zlog.Debug().Msgf("List inventory resources filter: %s", filter.String())

	if err := client.clientIsRegistered(); err != nil {
		return nil, err
	}

	object := inv_v1.ListResourcesRequest{
		ClientUuid: client.clientUUID,
		Filter:     filter,
	}
	objs, err := client.invAPI.ListResources(ctx, &object)
	if err != nil {
		zlog.Debug().Err(err).Msg("on List")
		return nil, inv_errors.Wrap(err)
	}

	return objs, nil
}

func (client *inventoryClient) ListAll(
	ctx context.Context,
	filter *inv_v1.Resource,
	fieldMask *fieldmaskpb.FieldMask,
) ([]*inv_v1.Resource, error) {
	zlog := zlog.TraceCtx(ctx)
	zlog.Debug().Msgf("List all inventory resources filter: %s", filter.String())
	if err := client.clientIsRegistered(); err != nil {
		return nil, err
	}

	filterRequest := inv_v1.ListResourcesRequest{
		ClientUuid: client.clientUUID,
		Filter: &inv_v1.ResourceFilter{
			Resource:  filter,
			FieldMask: fieldMask,
			Limit:     BatchSize,
			Offset:    0,
		},
	}
	resources := make([]*inv_v1.Resource, 0, BatchSize) // Pre-allocate a slice of at least a batchSize
	hasNext := true
	err := error(nil)
	for hasNext {
		var objs *inv_v1.ListResourcesResponse
		objs, err = client.invAPI.ListResources(ctx, &filterRequest)
		if inv_errors.IsNotFound(err) && len(resources) == 0 {
			return nil, err
		} else if err != nil {
			zlog.Debug().Err(err).Msg("on ListAll")
			// on errors, return partial result.
			// This covers also the case of interleaved deletes. In this case we could get a "not-found" error also when
			// getting a page different from the first.
			break
		}
		for _, r := range objs.GetResources() {
			resources = append(resources, r.GetResource())
		}
		hasNext = objs.HasNext
		filterRequest.Filter.Limit += BatchSize
		filterRequest.Filter.Offset += BatchSize
	}

	return removeDuplicates(resources), err
}

func (client *inventoryClient) Find(
	ctx context.Context,
	filter *inv_v1.ResourceFilter,
) (*inv_v1.FindResourcesResponse, error) {
	zlog := zlog.TraceCtx(ctx)
	zlog.Debug().Msgf("Find inventory resources filter: %s", filter.String())

	if err := client.clientIsRegistered(); err != nil {
		return nil, err
	}

	object := inv_v1.FindResourcesRequest{
		ClientUuid: client.clientUUID,
		Filter:     filter,
	}
	objs, err := client.invAPI.FindResources(ctx, &object)
	if err != nil {
		zlog.Debug().Err(err).Msg("on Find")
		return nil, inv_errors.Wrap(err)
	}

	return objs, nil
}

func (client *inventoryClient) FindAll(
	ctx context.Context,
	filter *inv_v1.Resource,
	fieldMask *fieldmaskpb.FieldMask,
) ([]string, error) {
	zlog := zlog.TraceCtx(ctx)
	zlog.Debug().Msgf("Find all inventory resources filter: %s", filter.String())
	if err := client.clientIsRegistered(); err != nil {
		return nil, err
	}

	filterRequest := inv_v1.FindResourcesRequest{
		ClientUuid: client.clientUUID,
		Filter: &inv_v1.ResourceFilter{
			Resource:  filter,
			FieldMask: fieldMask,
			Limit:     BatchSize,
			Offset:    0,
		},
	}
	resources := make([]string, 0, BatchSize) // Pre-allocate a slice of at least a batchSize
	hasNext := true
	err := error(nil)
	for hasNext {
		var objs *inv_v1.FindResourcesResponse
		objs, err = client.invAPI.FindResources(ctx, &filterRequest)
		if inv_errors.IsNotFound(err) && len(resources) == 0 {
			return nil, err
		} else if err != nil {
			zlog.Debug().Err(err).Msg("on FindAll")
			// on errors, return partial result.
			// This covers also the case of interleaved deletes. In this case we could get a "not-found" error also when
			// getting a page different from the first.
			break
		}
		resources = append(resources, objs.GetResourceId()...)
		hasNext = objs.HasNext
		filterRequest.Filter.Limit += BatchSize
		filterRequest.Filter.Offset += BatchSize
	}

	return removeDuplicates(resources), err
}

func (client *inventoryClient) Get(
	ctx context.Context,
	resourceID string,
) (*inv_v1.GetResourceResponse, error) {
	zlog.Debug().Msgf("Get inventory resource ID: %s", resourceID)

	if err := client.clientIsRegistered(); err != nil {
		return nil, err
	}

	object := inv_v1.GetResourceRequest{
		ClientUuid: client.clientUUID,
		ResourceId: resourceID,
	}
	obj, err := client.invAPI.GetResource(ctx, &object)
	if err != nil {
		zlog.Debug().Err(err).Msg("on Get")
		return nil, inv_errors.Wrap(err)
	}

	return obj, nil
}

func (client *inventoryClient) Create(
	ctx context.Context,
	res *inv_v1.Resource,
) (*inv_v1.CreateResourceResponse, error) {
	zlog.Debug().Msgf("Create inventory resource: %s", res.Resource)

	if err := client.clientIsRegistered(); err != nil {
		return nil, err
	}

	object := inv_v1.CreateResourceRequest{
		ClientUuid: client.clientUUID,
		Resource:   res,
	}
	obj, err := client.invAPI.CreateResource(ctx, &object)
	if err != nil {
		zlog.Debug().Err(err).Msg("on Create")
		invErr := client.handleInventoryError(err)
		return nil, invErr
	}

	return obj, nil
}

func (client *inventoryClient) Update(
	ctx context.Context,
	resourceID string,
	fieldmask *fieldmaskpb.FieldMask,
	res *inv_v1.Resource,
) (*inv_v1.UpdateResourceResponse, error) {
	zlog.Debug().Msgf("Update inventory resource: %s", res.Resource)

	if err := client.clientIsRegistered(); err != nil {
		return nil, err
	}

	object := inv_v1.UpdateResourceRequest{
		ClientUuid: client.clientUUID,
		ResourceId: resourceID,
		FieldMask:  fieldmask,
		Resource:   res,
	}
	obj, err := client.invAPI.UpdateResource(ctx, &object)
	if err != nil {
		zlog.Debug().Err(err).Msg("on Update")
		invErr := client.handleInventoryError(err)
		return nil, invErr
	}

	return obj, nil
}

func (client *inventoryClient) Delete(
	ctx context.Context,
	resourceID string,
) (*inv_v1.DeleteResourceResponse, error) {
	zlog.Debug().Msgf("Delete inventory resource ID: %s", resourceID)

	if err := client.clientIsRegistered(); err != nil {
		return nil, err
	}

	object := inv_v1.DeleteResourceRequest{
		ClientUuid: client.clientUUID,
		ResourceId: resourceID,
	}
	obj, err := client.invAPI.DeleteResource(ctx, &object)
	if err != nil {
		zlog.Debug().Err(err).Msg("on Delete")
		invErr := client.handleInventoryError(err)
		return nil, invErr
	}
	return obj, nil
}

// handleInventoryError handles errors returned by inventory.
// In particular, it handles the UNKNOWN_CLIENT error.
// It's currently applied to CREATE, UPDATE and DELETE methods,
// as these are the only methods that can modify the inventory state.
func (client *inventoryClient) handleInventoryError(err error) error {
	if inv_errors.IsUnKnownClient(err) {
		if client.cfg.AbortOnUnknownClientError {
			// Hotfix (see LPIO-1434)
			// In summary, sometimes RMs don't re-register to inventory after redeployment or restart.
			// As a consequence, RMs keep getting UNKNOWN_CLIENT error for update operations.
			// If this option is enabled, we can let RMs crash (and restart), so that they can re-register on startup.
			zlog.MiSec().Fatal().Msg("inventory client is unknown and abort on error enabled, aborting")
		} else {
			return inv_errors.Errorfc(codes.Unavailable,
				"inventory client is not registered: %s", err.Error())
		}
	}

	return inv_errors.Wrap(err)
}

// clientIsRegistered verifies if the client UUID is valid,
// i.e., if it is not invalid due to a subscription stream be closed.
func (client *inventoryClient) clientIsRegistered() error {
	client.uuidMutex.Lock()
	defer client.uuidMutex.Unlock()
	if client.clientUUID == "" {
		zlog.MiError("service unavailable - inventory client is not registered").Msg("")
		return inv_errors.Errorfc(codes.Unavailable, "inventory client is not registered")
	}
	return nil
}

func (client *inventoryClient) TestingOnlySetClient(c inv_v1.InventoryServiceClient) {
	client.invAPI = c
}

func removeDuplicates[T comparable](slice []T) []T {
	keys := make(map[T]struct{}, len(slice))
	noDupl := make([]T, 0, len(slice))
	for _, v := range slice {
		if _, ok := keys[v]; !ok {
			keys[v] = struct{}{}
			noDupl = append(noDupl, v)
		}
	}
	return noDupl
}
