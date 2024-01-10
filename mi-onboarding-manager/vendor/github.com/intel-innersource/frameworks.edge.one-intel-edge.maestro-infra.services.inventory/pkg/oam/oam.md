OAM gRPC Server
---------------------------

The OAM gRPC server implemented in this package offers a gRPC endpoint implementing the [gRPC health checking](https://github.com/grpc/grpc/blob/master/doc/health-checking.md) protocol. This together with the capabilities of k8s can be used to implement liveness/readiness meachanism for all our pods (not only for Inventory).

Each pod of the MI project has its own concept of readiness. For example the readiness of Inventory is reached only when the main gRPC server is ready to serve and the connection with the database is established.

## API Documentation

See the Go doc of the package for detailed function descriptions. Here is the
general workflow:

``` go
...
// Coordinate the readiness through this channel
readyChan = make(chan bool)
go func() {
    if err := oam.StartOamGrpcServer(termChan, readyChan, &wg, *oamservaddr, cfg.EnableTracing); err != nil {
        zlog.Fatal().Err(err).Msg("Cannot start Inventory OAM gRPC server")
    }
}()
...
// Signal to the OAM server that the service is now ready
// to serve clients as the store has been created
store := store.New()
...
// Note that on testing will be nil
if readyChan != nil {
    readyChan <- true
}
```

### Helm Chart Documentation

See a concrete example in the [values.yaml](https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.charts/blob/23dd86fcb3d0d8808a95a18f3c5e472792de77c4/mi-inventory/values.yaml#L42) of the Inventory charts where we have defined a cmd line parameter to configure the OAM server port.

Then we use the same endpoint to configure the gRPC health check client in the deployment configuration of the pod; see for example in the [statefulset.yaml](https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.charts/blob/23dd86fcb3d0d8808a95a18f3c5e472792de77c4/mi-inventory/templates/statefulset.yaml#L87) file of the Inventory charts.

Please note the same parameters are used to enable/disable the OAM gRPC server and the health check (they have to be both enabled or disabled).
