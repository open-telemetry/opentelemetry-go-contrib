package otlp

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"

	"go.opentelemetry.io/otel/log/global"
	otellog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func setupLog(ctx context.Context) (ShutdownFunc, error) {
	exporter, err := otlploghttp.New(ctx)

	if err != nil {
		return emptyShutdown, err
	}

	provider := otellog.NewLoggerProvider(
		otellog.WithResource(
			resource.Default(),
		),
		otellog.WithProcessor(
			otellog.NewBatchProcessor(exporter),
		),
	)

	global.SetLoggerProvider(provider)

	return provider.Shutdown, nil
}
