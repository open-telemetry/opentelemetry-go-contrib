// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// Supported protocols for OTLP exporter.
	protocolProtobufHTTP = "http/protobuf"
	protocolProtobufGRPC = "grpc/protobuf"
)

var (
	errInvalidExporterConfiguration = fmt.Errorf("invalid exporter configuration")
	errUnsupportedSpanProcessorType = fmt.Errorf("unsupported span processor type")
	errUnsupportedMetricReaderType  = fmt.Errorf("unsupported metric reader type")
)

// Validate checks for a valid batch processor for the SpanProcessor.
func (sp *SpanProcessor) Validate() error {
	if sp.Batch != nil {
		return sp.Batch.Exporter.Validate()
	}
	return errUnsupportedSpanProcessorType
}

// Validate checks for valid exporters to be configured for the SpanExporter.
func (se *SpanExporter) Validate() error {
	if se.Console == nil && se.Otlp == nil {
		return errInvalidExporterConfiguration
	}
	return nil
}

// Validate checks the configuration for Prometheus exporter.
func (p *Prometheus) Validate() error {
	if p.Host == nil {
		return fmt.Errorf("host must be specified")
	}
	if p.Port == nil {
		return fmt.Errorf("port must be specified")
	}
	return nil
}

// Validate checks the configuration for OtlpMetric exporter.
func (om *OtlpMetric) Validate() error {
	switch om.Protocol {
	case protocolProtobufHTTP:
	case protocolProtobufGRPC:
	default:
		return fmt.Errorf("unsupported protocol %s", om.Protocol)
	}

	if len(om.Endpoint) > 0 {
		_, err := url.ParseRequestURI(normalizeEndpoint(om.Endpoint))
		if err != nil {
			return err
		}
	}
	if om.Compression != nil {
		switch *om.Compression {
		case "gzip":
		case "none":
		default:
			return fmt.Errorf("unsupported compression %q", *om.Compression)
		}
	}
	return nil
}

// Validate checks for either a valid pull or periodic exporter for the MetricReader.
func (mr *MetricReader) Validate() error {
	if mr.Pull != nil {
		return mr.Pull.Validate()
	}
	if mr.Periodic != nil {
		return mr.Periodic.Validate()
	}

	return errUnsupportedMetricReaderType
}

// Validate checks for valid exporters to be configured for the PullMetricReader.
func (pmr *PullMetricReader) Validate() error {
	if pmr.Exporter.Prometheus == nil {
		return errInvalidExporterConfiguration
	}
	return pmr.Exporter.Validate()
}

// Validate calls the configured exporter's Validate method.
func (me *MetricExporter) Validate() error {
	if me.Otlp != nil {
		return me.Otlp.Validate()
	}
	if me.Console != nil {
		return nil
	}
	if me.Prometheus != nil {
		return me.Prometheus.Validate()
	}
	return errInvalidExporterConfiguration
}

// Validate checks for valid exporters to be configured for the PeriodicMetricReader.
func (pmr *PeriodicMetricReader) Validate() error {
	if pmr.Exporter.Otlp == nil && pmr.Exporter.Console == nil {
		return errInvalidExporterConfiguration
	}
	return pmr.Exporter.Validate()
}

// Validate checks for a valid Selector or Stream to be configured for the View.
func (v *View) Validate() error {
	if v.Selector == nil || v.Stream == nil {
		return fmt.Errorf("invalid view configuration")
	}
	return nil
}

func (s *ViewSelector) instrumentNameStr() string {
	if s.InstrumentName == nil {
		return ""
	}
	return *s.InstrumentName
}

func (s *ViewSelector) meterNameStr() string {
	if s.MeterName == nil {
		return ""
	}
	return *s.MeterName
}

func (s *ViewSelector) meterVersionStr() string {
	if s.MeterVersion == nil {
		return ""
	}
	return *s.MeterVersion
}

func (s *ViewSelector) meterSchemaURLStr() string {
	if s.MeterSchemaUrl == nil {
		return ""
	}
	return *s.MeterSchemaUrl
}

func (s *ViewSelector) unitStr() string {
	if s.Unit == nil {
		return ""
	}
	return *s.Unit
}

func (s *ViewStream) nameStr() string {
	if s.Name == nil {
		return ""
	}
	return *s.Name
}

func (s *ViewStream) descriptionStr() string {
	if s.Description == nil {
		return ""
	}
	return *s.Description
}

func (e *ViewStreamAggregationExplicitBucketHistogram) recordMinMaxBool() bool {
	if e.RecordMinMax == nil {
		return false
	}
	return *e.RecordMinMax
}

func normalizeEndpoint(endpoint string) string {
	if !strings.HasPrefix(endpoint, "https://") && !strings.HasPrefix(endpoint, "http://") {
		return fmt.Sprintf("http://%s", endpoint)
	}
	return endpoint
}
