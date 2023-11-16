// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	errNoValidSpanExporter = errors.New("no valid span exporter")
)

func initTracerProvider(cfg configOptions) (trace.TracerProvider, shutdownFunc) {
	if cfg.opentelemetryConfig.TracerProvider == nil {
		return noop.NewTracerProvider(), noopShutdown
	}

	// TODO: add support for:
	// - cfg.opentelemetryConfig.TracerProvider.Limits
	// - cfg.opentelemetryConfig.TracerProvider.Sampler
	opts := []sdktrace.TracerProviderOption{}
	var processor sdktrace.SpanProcessor
	var err error
	for _, sp := range cfg.opentelemetryConfig.TracerProvider.Processors {
		processor, err = initSpanProcessor(context.Background(), sp)
		if err != nil {
			// TODO: handler err
			panic(err)
		}
		opts = append(opts, sdktrace.WithSpanProcessor(processor))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	return tp, tp.Shutdown
}

// func (sp *SpanProcessor) Unmarshal(conf *confmap.Conf) error {
// 	if !obsreportconfig.UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate.IsEnabled() {
// 		// only unmarshal if feature gate is enabled
// 		return nil
// 	}

// 	if conf == nil {
// 		return nil
// 	}

// 	if err := conf.Unmarshal(sp); err != nil {
// 		return fmt.Errorf("invalid span processor configuration: %w", err)
// 	}

// 	if sp.Batch != nil {
// 		return sp.Batch.Exporter.Validate()
// 	}
// 	return fmt.Errorf("unsupported span processor type %s", conf.AllKeys())
// }

// Validate checks for valid exporters to be configured for the SpanExporter
func (se *SpanExporter) Validate() error {
	if se.Console == nil && se.OTLP == nil {
		return fmt.Errorf("invalid exporter configuration")
	}
	return nil
}

func initSpanProcessor(ctx context.Context, processor SpanProcessor) (sdktrace.SpanProcessor, error) {
	if processor.Batch != nil {
		if processor.Batch.Exporter.Console != nil {
			exp, err := stdouttrace.New(
				stdouttrace.WithPrettyPrint(),
			)
			if err != nil {
				return nil, err
			}
			return initBatchSpanProcessor(processor.Batch, exp)
		}
		// 		if processor.Batch.Exporter.Otlp != nil {
		// 			var err error
		// 			var exp sdktrace.SpanExporter
		// 			switch processor.Batch.Exporter.Otlp.Protocol {
		// 			case protocolProtobufHTTP:
		// 				exp, err = initOTLPHTTPSpanExporter(ctx, processor.Batch.Exporter.Otlp)
		// 			case protocolProtobufGRPC:
		// 				exp, err = initOTLPgRPCSpanExporter(ctx, processor.Batch.Exporter.Otlp)
		// 			default:
		// 				return nil, fmt.Errorf("unsupported protocol %q", processor.Batch.Exporter.Otlp.Protocol)
		// 			}
		// 			if err != nil {
		// 				return nil, err
		// 			}
		// 			return initBatchSpanProcessor(processor.Batch, exp)
		// 		}
		return nil, errNoValidSpanExporter
	}
	return nil, fmt.Errorf("unsupported span processor type %v", processor)
}

func initBatchSpanProcessor(bsp *BatchSpanProcessor, exp sdktrace.SpanExporter) (sdktrace.SpanProcessor, error) {
	opts := []sdktrace.BatchSpanProcessorOption{}
	if bsp.ExportTimeout != nil {
		if *bsp.ExportTimeout < 0 {
			return nil, fmt.Errorf("invalid export timeout %d", *bsp.ExportTimeout)
		}
		opts = append(opts, sdktrace.WithExportTimeout(time.Millisecond*time.Duration(*bsp.ExportTimeout)))
	}
	if bsp.MaxExportBatchSize != nil {
		if *bsp.MaxExportBatchSize < 0 {
			return nil, fmt.Errorf("invalid batch size %d", *bsp.MaxExportBatchSize)
		}
		opts = append(opts, sdktrace.WithMaxExportBatchSize(*bsp.MaxExportBatchSize))
	}
	if bsp.MaxQueueSize != nil {
		if *bsp.MaxQueueSize < 0 {
			return nil, fmt.Errorf("invalid queue size %d", *bsp.MaxQueueSize)
		}
		opts = append(opts, sdktrace.WithMaxQueueSize(*bsp.MaxQueueSize))
	}
	if bsp.ScheduleDelay != nil {
		if *bsp.ScheduleDelay < 0 {
			return nil, fmt.Errorf("invalid schedule delay %d", *bsp.ScheduleDelay)
		}
		opts = append(opts, sdktrace.WithBatchTimeout(time.Millisecond*time.Duration(*bsp.ScheduleDelay)))
	}
	return sdktrace.NewBatchSpanProcessor(exp, opts...), nil

}
