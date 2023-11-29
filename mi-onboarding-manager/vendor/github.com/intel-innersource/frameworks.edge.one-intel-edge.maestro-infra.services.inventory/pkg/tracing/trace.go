// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tracing

import (
	"context"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

var log = logging.GetLogger("tracing")

func newTraceResources(service string, attribs map[string]string) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{
		semconv.ServiceName(service),
	}

	for attribKey, attribValue := range attribs {
		attributes = append(attributes, attribute.String(attribKey, attribValue))
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attributes...,
		),
	)
	if err != nil {
		log.Warn().Err(err).Msg("Could not set trace resources")
	}

	return resources, errors.Wrap(err)
}

func newTraceExporter(client otlptrace.Client, resources *resource.Resource) (func(context.Context) error, error) {
	exporter, err := otlptrace.New(
		context.Background(),
		client,
	)
	if err != nil {
		log.Warn().Err(err).Msg("Could not create trace exporter")
		return nil, errors.Wrap(err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithResource(resources),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithSpanProcessor(bsp),
		),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return exporter.Shutdown, nil
}

// NewTraceExporterHTTP creates and starts a new exporter of traces to the provided otelURL using HTTP
// The tracing definition is annotated with the service string attribute.
// Other trace attributes can be provided by the attribs map.
// The function returns a function to call the shutdown of the trace.
func NewTraceExporterHTTP(otelURL, service string, attribs map[string]string) (func(context.Context) error, error) {
	secureOption := otlptracehttp.WithInsecure()
	traceClient := otlptracehttp.NewClient(
		secureOption,
		otlptracehttp.WithEndpoint(otelURL),
	)
	resources, err := newTraceResources(service, attribs)
	if err != nil {
		return nil, err
	}
	exporterShutdown, err := newTraceExporter(traceClient, resources)
	if err != nil {
		return nil, err
	}

	return exporterShutdown, nil
}

// NewTraceExporterGRPC creates and start a new exporter of traces to the provided otelgrpcURL using gRPC
// The tracing definition is annotated with the service string attribute.
// Other trace attributes can be provided by the attribs map.
// The function returns a function to call the shutdown of the trace.
func NewTraceExporterGRPC(otelgrpcURL, service string, attribs map[string]string) (func(context.Context) error, error) {
	secureOption := otlptracegrpc.WithInsecure()
	traceClient := otlptracegrpc.NewClient(
		secureOption,
		otlptracegrpc.WithEndpoint(otelgrpcURL),
	)

	resources, err := newTraceResources(service, attribs)
	if err != nil {
		return nil, err
	}
	exporterShutdown, err := newTraceExporter(traceClient, resources)
	if err != nil {
		return nil, err
	}

	return exporterShutdown, nil
}

// StartTrace starts a new trace and a child span of it.
// It must be used with StopTrace function to end the child span.
// StopTrace can be used with defer statement after StartTrace call.
func StartTrace(ctx context.Context, servicename, traceName string) context.Context {
	ctx, _ = otel.Tracer(servicename).Start(ctx, traceName)
	return ctx
}

func StartTraceFromRemote(ctx context.Context, servicename, traceName string) context.Context {
	tracer := otel.Tracer(servicename)
	ctx, _ = tracer.Start(
		trace.ContextWithRemoteSpanContext(ctx, trace.SpanContextFromContext(ctx)),
		traceName,
		trace.WithSpanKind(trace.SpanKindClient),
	)
	return ctx
}

// StopTrace ends a span trace from the provided context.
// StopTrace can be used with defer statement after StartTrace call.
func StopTrace(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	span.End()
}

// EnableGrpcClientTracing adds automated tracing instrumentation
// to client options.
func EnableGrpcClientTracing(opts []grpc.DialOption) []grpc.DialOption {
	return append(opts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
}

// EnableGrpcServerTracing adds automated tracing instrumentation
// to server options.
func EnableGrpcServerTracing(opts []grpc.ServerOption) []grpc.ServerOption {
	return append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
}

// EnableEchoAutoTracing adds opentelemetry middleware
// to echo server for automated tracing instrumentation.
func EnableEchoAutoTracing(e *echo.Echo, name string) {
	e.Use(otelecho.Middleware(name))
}
