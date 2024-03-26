package main

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"math/rand"
	"os"
	"time"
)

func SetupOtepConfig() {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://3.85.189.32:4317/")
	os.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
}

func SetupTraceProvider() *trace.TracerProvider {
	ctx := context.Background()
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		panic(err)
	}

	provider := trace.NewTracerProvider(
		trace.WithSpanProcessor(
			trace.NewSimpleSpanProcessor(exporter),
		),
	)

	otel.SetTracerProvider(provider)

	return provider
}

func SetupMetricProvider() *metric.MeterProvider {
	ctx := context.Background()
	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		panic(err)
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(
			metric.NewPeriodicReader(exporter),
		),
	)

	otel.SetMeterProvider(provider)

	return provider
}

func Service(ctx context.Context, in any) (any, error) {
	tracer := otel.Tracer("Service-Trace")
	meter := otel.Meter("Service-Meter")

	fail, err := meter.Int64Counter("failed")
	if err != nil {
		return nil, err
	}

	success, err := meter.Int64Counter("success")
	if err != nil {
		return nil, err
	}

	trx, span := tracer.Start(ctx, "Service.Process")
	defer span.End()

	req := in.(map[string]any)
	if v, ok := req["error"]; ok {
		fail.Add(trx, 1)
		span.SetStatus(codes.Error, "error")
		return nil, fmt.Errorf("%v", v)
	}

	success.Add(trx, 1)
	span.SetStatus(codes.Ok, "success")

	return req, nil
}

func main() {
	ctx := context.Background()

	SetupOtepConfig()

	// ADD Trace
	{
		p := SetupTraceProvider()
		defer func() {
			if err := p.Shutdown(ctx); err != nil {
				panic(err)
			}
		}()
	}

	// ADD Meter
	{
		p := SetupMetricProvider()
		defer func() {
			if err := p.Shutdown(ctx); err != nil {
				panic(err)
			}
		}()
	}

	req := []any{
		map[string]any{
			"foo": "bar",
		},
		map[string]any{
			"error": "an any error",
		},
	}

	for i := 0; i < 1000; i++ {
		Service(ctx, req[rand.Intn(2)%2])
		time.Sleep(10 * time.Millisecond)
	}
}
