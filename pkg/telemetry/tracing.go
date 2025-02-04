package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

func NewTraceExporter(ctx context.Context, conn *grpc.ClientConn) (*otlptrace.Exporter, error) {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return exporter, fmt.Errorf("error creatin new exporter: %w", err)
	}

	return exporter, nil
}

func NewResource(ctx context.Context, applicationName string) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithContainer(),
		resource.WithAttributes(
			attribute.String("service.name", applicationName),
		),
	)
	if err != nil {
		return res, fmt.Errorf("error createing new resource %w", err)
	}

	return res, nil
}

func NewTraceProvider(res *resource.Resource, bsp sdktrace.SpanProcessor) *sdktrace.TracerProvider {
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	return tracerProvider
}
