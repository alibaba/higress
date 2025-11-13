package observability

import (
	context "context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/sdk/trace"
)

// SetupTracing configures a global tracer provider exporting to an OTLP collector over HTTP.
func SetupTracing(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter, trace.WithMaxExportBatchSize(8192), trace.WithBatchTimeout(3*time.Second)),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}