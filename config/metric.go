// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

var errNoValidMetricExporter = errors.New("no valid metric exporter")

func initMeterProvider(cfg configOptions, res *resource.Resource) (metric.MeterProvider, shutdownFunc, error) {
	if cfg.opentelemetryConfig.MeterProvider == nil {
		return noop.NewMeterProvider(), noopShutdown, nil
	}

	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
		// TODO: configure views
		// sdkmetric.WithView(batchViews(disableHighCardinality)...),
	}

	for _, reader := range cfg.opentelemetryConfig.MeterProvider.Readers {
		r, err := initMetricReader(context.Background(), reader)
		if err != nil {
			return noop.NewMeterProvider(), noopShutdown, nil
		}
		opts = append(opts, sdkmetric.WithReader(r))
	}
	mp := sdkmetric.NewMeterProvider(opts...)
	return mp, mp.Shutdown, nil
}

func initPrometheusExporter(prometheusConfig *Prometheus) (sdkmetric.Reader, error) {
	if prometheusConfig.Host == nil {
		return nil, fmt.Errorf("host must be specified")
	}
	if prometheusConfig.Port == nil {
		return nil, fmt.Errorf("port must be specified")
	}
	exporter, err := otelprom.New(
	//
	// https://github.com/open-telemetry/opentelemetry-configuration/pull/61
	// otelprom.WithoutUnits(),
	// otelprom.WithoutCounterSuffixes(),
	//
	// TODO: how are these options configured?
	// otelprom.WithRegisterer(prometheus.NewRegistry()),
	// otelprom.WithoutScopeInfo(),
	// otelprom.WithProducer(opencensus.NewMetricProducer()),
	// otelprom.WithNamespace("otelcol"),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating otel prometheus exporter: %w", err)
	}

	return exporter, nil
}

func initPullExporter(exporter MetricExporter) (sdkmetric.Reader, error) {
	if exporter.Prometheus != nil {
		return initPrometheusExporter(exporter.Prometheus)
	}
	return nil, errNoValidMetricExporter
}

func initPeriodicExporter(ctx context.Context, exporter MetricExporter, opts ...sdkmetric.PeriodicReaderOption) (sdkmetric.Reader, error) {
	if exporter.Console != nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		exp, err := stdoutmetric.New(
			stdoutmetric.WithEncoder(enc),
		)
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, opts...), nil
	}
	if exporter.OTLP != nil {
		var err error
		var exp sdkmetric.Exporter
		switch exporter.OTLP.Protocol {
		case protocolProtobufHTTP:
			exp, err = initOTLPHTTPExporter(ctx, exporter.OTLP)
		case protocolProtobufGRPC:
			exp, err = initOTLPgRPCExporter(ctx, exporter.OTLP)
		default:
			return nil, fmt.Errorf("unsupported protocol %s", exporter.OTLP.Protocol)
		}
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, opts...), nil
	}
	return nil, errNoValidMetricExporter
}

func initMetricReader(ctx context.Context, reader MetricReader) (sdkmetric.Reader, error) {
	if reader.Pull != nil {
		return initPullExporter(reader.Pull.Exporter)
	}
	if reader.Periodic != nil {
		opts := []sdkmetric.PeriodicReaderOption{}
		if reader.Periodic.Interval != nil {
			opts = append(opts, sdkmetric.WithInterval(time.Duration(*reader.Periodic.Interval)*time.Millisecond))
		}

		if reader.Periodic.Timeout != nil {
			opts = append(opts, sdkmetric.WithTimeout(time.Duration(*reader.Periodic.Timeout)*time.Millisecond))
		}
		return initPeriodicExporter(ctx, reader.Periodic.Exporter, opts...)
	}
	return nil, fmt.Errorf("unsupported metric reader type %v", reader)
}

func initOTLPgRPCExporter(ctx context.Context, otlpConfig *OTLPMetric) (sdkmetric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(normalizeEndpoint(otlpConfig.Endpoint))
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetricgrpc.WithEndpoint(u.Host))
		if u.Scheme == "http" {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
	}

	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case "gzip":
			opts = append(opts, otlpmetricgrpc.WithCompressor(*otlpConfig.Compression))
		case "none":
			// none requires no options
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil {
		opts = append(opts, otlpmetricgrpc.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(otlpConfig.Headers))
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

func initOTLPHTTPExporter(ctx context.Context, otlpConfig *OTLPMetric) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{}

	if len(otlpConfig.Endpoint) > 0 {
		u, err := url.ParseRequestURI(normalizeEndpoint(otlpConfig.Endpoint))
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetrichttp.WithEndpoint(u.Host))

		if u.Scheme == "http" {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(u.Path) > 0 {
			opts = append(opts, otlpmetrichttp.WithURLPath(u.Path))
		}
	}
	if otlpConfig.Compression != nil {
		switch *otlpConfig.Compression {
		case "gzip":
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
		case "none":
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.NoCompression))
		default:
			return nil, fmt.Errorf("unsupported compression %q", *otlpConfig.Compression)
		}
	}
	if otlpConfig.Timeout != nil {
		opts = append(opts, otlpmetrichttp.WithTimeout(time.Millisecond*time.Duration(*otlpConfig.Timeout)))
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(otlpConfig.Headers))
	}

	return otlpmetrichttp.New(ctx, opts...)
}
