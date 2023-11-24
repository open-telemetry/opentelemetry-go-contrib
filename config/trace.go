// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var errNoValidSpanExporter = errors.New("no valid span exporter")

func initTracerProvider(cfg configOptions, res *resource.Resource) (trace.TracerProvider, shutdownFunc, error) {
	if cfg.opentelemetryConfig.TracerProvider == nil {
		return noop.NewTracerProvider(), noopShutdown, nil
	}
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}
	for _, processor := range cfg.opentelemetryConfig.TracerProvider.Processors {
		sp, err := initSpanProcessor(cfg.ctx, processor)
		if err != nil {
			return noop.NewTracerProvider(), noopShutdown, err
		}
		opts = append(opts, sdktrace.WithSpanProcessor(sp))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	return tp, tp.Shutdown, nil
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
		if processor.Batch.Exporter.OTLP != nil {
			var err error
			var exp sdktrace.SpanExporter
			switch processor.Batch.Exporter.OTLP.Protocol {
			case protocolProtobufHTTP:
				exp, err = initOTLPHTTPSpanExporter(ctx, processor.Batch.Exporter.OTLP)
			case protocolProtobufGRPC:
				exp, err = initOTLPgRPCSpanExporter(ctx, processor.Batch.Exporter.OTLP)
			default:
				return nil, fmt.Errorf("unsupported protocol %q", processor.Batch.Exporter.OTLP.Protocol)
			}
			if err != nil {
				return nil, err
			}
			return initBatchSpanProcessor(processor.Batch, exp)
		}
		return nil, errNoValidSpanExporter
	}
	return nil, fmt.Errorf("unsupported span processor type %v", processor)
}

func initOTLPgRPCSpanExporter(ctx context.Context, otlpConfig *OTLP) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(normalizeEndpoint(otlpConfig.Endpoint))
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracegrpc.WithEndpoint(u.Host))
		if u.Scheme == "http" {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
	}

	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case "gzip":
			opts = append(opts, otlptracegrpc.WithCompressor(*otlpConfig.Compression))
		case "none":
			// none requires no options
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlptracegrpc.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(otlpConfig.Headers))
	}

	return otlptracegrpc.New(ctx, opts...)
}

func initOTLPHTTPSpanExporter(ctx context.Context, otlpConfig *OTLP) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(normalizeEndpoint(otlpConfig.Endpoint))
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracehttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(u.Path) > 0 {
			opts = append(opts, otlptracehttp.WithURLPath(u.Path))
		}
	}
	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case "gzip":
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
		case "none":
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.NoCompression))
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil && *otlpConfig.Timeout > 0 {
		opts = append(opts, otlptracehttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(otlpConfig.Headers))
	}

	return otlptracehttp.New(ctx, opts...)
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
