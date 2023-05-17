// Package tracing package
package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serviceName     = "tharsis-api"
	gRPCDialTimeout = 30 * time.Second
)

type traceType string

const (
	otlpTraceType traceType = "otlp"
	xrayTraceType traceType = "xray"
)

// NewProviderInput holds fields to create a new provider.
type NewProviderInput struct {
	Type    string
	Host    string
	Version string
	Port    int
	Enabled bool
}

// NewProvider initializes the global/default trace provider.
func NewProvider(ctx context.Context, input *NewProviderInput) (func(context.Context) error, error) {

	if !input.Enabled {
		// If disabled, default to the no-op provider with a no-op shutdown function.
		return func(context.Context) error { return nil }, nil
	}

	// Make sure the trace type is valid.
	checkedType := traceType(input.Type)
	switch checkedType {
	case otlpTraceType, xrayTraceType:
	default:
		return nil, fmt.Errorf("invalid trace type: %s", input.Type)
	}

	exp, err := newExporter(ctx, input.Host, input.Port)
	if err != nil {
		return nil, err
	}

	tp := newTracerProvider(checkedType, exp, newResource(input.Version))
	otel.SetTracerProvider(tp)

	// Documentation says default global propagator is no-op.
	if checkedType == xrayTraceType {
		otel.SetTextMapPropagator(xray.Propagator{})
	} else {
		otel.SetTextMapPropagator(propagation.TraceContext{})
	}

	return tp.Shutdown, nil
}

func newResource(serviceVersion string) *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	return r
}

func newExporter(ctx context.Context, host string, port int) (sdktrace.SpanExporter, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, gRPCDialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctxWithTimeout,
		fmt.Sprintf("%s:%d", host, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open gRPC connection: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	return exporter, nil
}

func newTracerProvider(traceType traceType, exp sdktrace.SpanExporter, res *resource.Resource) *sdktrace.TracerProvider {
	options := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exp)),
	}

	// X-Ray tracing requires a particular format for the trace ID.
	if traceType == xrayTraceType {
		options = append(options, sdktrace.WithIDGenerator(xray.NewIDGenerator()))
	}

	return sdktrace.NewTracerProvider(options...)
}

// RecordError is a convenience function for recording an error and setting span status.
func RecordError(span trace.Span, err error, format string, args ...any) {

	// If there is no pre-defined error object, make one from the description.
	if err == nil {
		err = fmt.Errorf(format, args...)
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, fmt.Sprintf(format, args...))
}
