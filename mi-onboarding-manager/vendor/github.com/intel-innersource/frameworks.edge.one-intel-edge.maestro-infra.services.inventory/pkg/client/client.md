Inventory gRPC Client
---------------------------

The Inventory gRPC client implemented in this package offers a convenient
wrapper around the raw inventory gRPC API. It can be used by resource managers
and other consumers of that API to get an easy, safe, and simplified interface.

## API Documentation

See the Go doc of the package for detailed function descriptions. Here is the
general workflow:

```go

// You can also create a new client with unary interceptors as follows.
unaryInterceptors := []grpc.UnaryClientInterceptor{
    retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable))
}

// A client needs to be instantiated with a configuration, for instance:
clientCfg := client.InventoryClientConfig{
    Name:    "test_client",
    Address: "bufconn",
    SecurityCfg: &client.SecurityConfig{
        Insecure: false,
        CaPath:   "/cert/certificates/ca-cert.pem",
        CertPath: "/cert/certificates/client-cert.pem",
        KeyPath:  "/cert/certificates/client-key.pem",
    },
    Events:            make(chan *client.WatchEvents, eventsBufSize),
    ClientKind:        inv_v1.ClientKind_CLIENT_KIND_RESOURCE_MANAGER,
    ResourceKinds:     []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE},
    TermChan:          termChan,
    Wg:                wg,
    EnableTracing:     true,
    UnaryInterceptors: unaryInterceptors,
}

// Create a new client. It will be automatically registered.
gcli, err := NewInventoryClient(context.Background(), clientCfg)
defer gcli.Close()


// Resources can be retrieved with the Get function.
resp, err := gcli.Get(ctx, "host-1234567")
host := resp.GetHost()

// EventChannel provides access to the received inventory events that this client
// subscribed to.
// The Events provided by the configuration of the client is the same returned by EventChannel().
ev := <-gcli.EventChannel()
ev.EventKind // SubscribeEventsResponse_EVENT_KIND_CREATED
ev.ResourceId // host-1234567

// The client interface provides functions to Create/Get/Update/Delete/Find/List resources.
// For example:
createresreq := &inv_v1.Resource{
    Resource: &inv_v1.Resource_Instance{Instance: &vmres},
}

// build a context for gRPC
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

createresresp, err := gcli.Create(ctx, createresreq)
if err != nil {
    return "", err
}

// Clients should call Close() to initiate orderly shutdown.
err = gcli.Close()
```

Notice, `InventoryClientConfig` allows two parameters to be configured:

```go
	EnableRegisterRetry    bool
	RegisterMaxElapsedTime int
```

If set to 'true' EnableRegisterRetry enables a goroutine to keep retrying to register
the inventory client until a RegisterMaxElapsedTime (seconds) is reached.
Any errors in the registration retry loop are logged as warnings.

Important, the registration retry routine does not guarantee the consistency or
order of the subscription Events received by the client.
The client needs to make sure to retrieve this gap,
once the stream is reestablished with an initial/periodic reconciliation.