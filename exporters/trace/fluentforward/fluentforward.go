// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fluentforward

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/vmihailenco/msgpack/v5"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/label"
	export "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// ffSpan is serialized as a messagepack array of the form:
//
// [
// 	"tag",
// 	"timestamp",
// 	{"key": "value", ...}
// ]

type ffSpan struct {
	_msgpack struct{} `msgpack:",asArray"`
	Tag      string   `msgpack:"tag"`
	Ts       int64    `msgpack:"ts"`
	SpanData SpanData `msgpack:"spanData"`
}

// Refer https://github.com/open-telemetry/opentelemetry-proto/blob/master/opentelemetry/proto/trace/v1/trace.proto
// for a verbose description of the fields

// SpanData contains all the properties of the span.
type SpanData struct {
	TraceID                       string                    `msgpack:"traceId"` // A unique identifier for the trace
	SpanID                        string                    `msgpack:"spanId"`  // A unique identifier for a span within a trace
	ParentSpanID                  string                    `msgpack:"parentSpanId"`
	Name                          string                    `msgpack:"name"`                   // A description of the spans operation
	StartTime                     int64                     `msgpack:"startTime"`              // Start time of the span
	EndTime                       int64                     `msgpack:"endTime"`                // End time of the span
	Attrs                         map[label.Key]interface{} `msgpack:"attrs"`                  // A collection of key-value pairs
	DroppedAttributeCount         int                       `msgpack:"droppedAttributesCount"` // Number of attributes that were dropped due to reasons like too many attributes
	Links                         []Link                    `msgpack:"links"`
	DroppedLinkCount              int                       `msgpack:"droppedLinkCount"`
	StatusCode                    string                    `msgpack:"statusCode"` // Status code of the span. Defaults to unset
	MessageEvents                 []Event                   `msgpack:"messageEvents"`
	DroppedMessageEventCount      int                       `msgpack:"droppedMessageEventCount"`
	SpanKind                      trace.SpanKind            `msgpack:"spanKind"`                   // Type of span
	StatusMessage                 string                    `msgpack:"statusMessage"`              // Human readable error message
	InstrumentationLibraryName    string                    `msgpack:"instrumentationLibraryName"` // Instrumentation library used to provide instrumentation
	InstrumentationLibraryVersion string                    `msgpack:"instrumentationLibraryVersion"`
	Resource                      string                    `msgpack:"resource"` // Contains attributes representing an entity that produced this span
}

// An event is a time-stamped annotation of the span that has user supplied text description and key-value pairs
type Event struct {
	Ts    int64                     `msgpack:"ts"`    // The time at which the event occurred
	Name  string                    `msgpack:"name"`  // Event name
	Attrs map[label.Key]interface{} `msgpack:"attrs"` // collection of key-value pairs on the event
}

// A link contains references from this span to a span in the same or different trace
type Link struct {
	TraceID string                    `msgpack:"traceId"`
	SpanID  string                    `msgpack:"spanId"`
	Attrs   map[label.Key]interface{} `msgpack:"attrs"`
}

// Exporter implements the SpanExporter interface that allows us to export span data
type Exporter struct {
	url         string
	serviceName string
	client      *reconnectingTCPConn
	o           options
}

// Option defines a function that configures the exporter.
type Option func(*options)

// Options contains configuration for the exporter.
type options struct {
	config *sdktrace.Config
	logger *log.Logger
}

// WithLogger configures the exporter to use the passed logger.
func WithLogger(logger *log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithSDK sets the SDK config for the exporter pipeline.
func WithSDK(config *sdktrace.Config) Option {
	return func(o *options) {
		o.config = config
	}
}

// InstallNewPipeline instantiates a NewExportPipeline with the
// recommended configuration and registers it globally.
func InstallNewPipeline(ffurl, serviceName string, opts ...Option) error {
	tp, err := NewExportPipeline(ffurl, serviceName, opts...)
	if err != nil {
		return err
	}
	otel.SetTracerProvider(tp)
	return nil
}

// NewExportPipeline sets up a complete export pipeline
// with the recommended setup for trace provider
func NewExportPipeline(ffurl, serviceName string, opts ...Option) (*sdktrace.TracerProvider, error) {
	exp, err := NewRawExporter(ffurl, serviceName, opts...)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp))
	if exp.o.config != nil {
		tp.ApplyConfig(*exp.o.config)
	}
	return tp, nil
}

// NewRawExporter creates a new exporter
func NewRawExporter(ffurl, serviceName string, opts ...Option) (*Exporter, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	if ffurl == "" {
		return nil, errors.New("fluent instance url cannot be empty")
	}

	client, err := newReconnectingTCPConn(ffurl, 10*time.Second, net.ResolveTCPAddr, net.DialTCP)
	if err != nil {
		return nil, err
	}

	return &Exporter{
		url:         ffurl,
		serviceName: serviceName,
		client:      client,
		o:           o,
	}, nil

}

// Export spans to fluent instance
func (e *Exporter) ExportSpans(ctx context.Context, sds []*export.SpanData) error {

	for _, span := range sds {
		var s struct{}
		ffspan := ffSpan{
			_msgpack: s,
			Tag:      "span.test",
			Ts:       span.EndTime.UnixNano(),
		}

		spans := SpanData{}
		spans.TraceID = span.SpanContext.TraceID.String()
		spans.SpanID = span.SpanContext.SpanID.String()
		spans.ParentSpanID = span.ParentSpanID.String()
		spans.SpanKind = span.SpanKind
		spans.Name = span.Name
		spans.StatusMessage = span.StatusMessage
		spans.StatusCode = span.StatusCode.String()
		spans.StartTime = span.StartTime.UnixNano()
		spans.EndTime = span.EndTime.UnixNano()
		spans.InstrumentationLibraryName = span.InstrumentationLibrary.Name
		spans.InstrumentationLibraryVersion = span.InstrumentationLibrary.Version
		spans.Resource = span.Resource.String()

		spans.MessageEvents = eventsToSlice(span.MessageEvents)
		spans.DroppedMessageEventCount = span.DroppedMessageEventCount

		spans.Attrs = attributesToMap(span.Attributes)
		spans.DroppedAttributeCount = span.DroppedAttributeCount

		spans.Links = linksToSlice(span.Links)
		spans.DroppedLinkCount = span.DroppedLinkCount

		ffspan.SpanData = spans

		t, err := msgpack.Marshal(&ffspan)
		if err != nil {
			return errors.New("unable to serialize span data")
		}

		_, err = e.client.Write(t)
		if err != nil {
			return fmt.Errorf("error while writing to %s: %v", e.url, err)
		}
	}
	return nil
}

// Stop the exporter
func (e *Exporter) Shutdown(ctx context.Context) error {
	e.client.Close()
	return nil
}

// attributesToMap converts attributes from a slice of key-values to a map for exporting
func attributesToMap(attributes []label.KeyValue) map[label.Key]interface{} {
	attrs := make(map[label.Key]interface{})
	for _, v := range attributes {
		attrs[v.Key] = v.Value.AsInterface()
	}
	return attrs
}

// linksToSlice converts links from the format []trace.Link to []Link for exporting
func linksToSlice(links []trace.Link) []Link {
	var l []Link
	for _, v := range links {
		temp := Link{
			TraceID: v.SpanContext.TraceID.String(),
			SpanID:  v.SpanContext.SpanID.String(),
			Attrs:   attributesToMap(v.Attributes),
		}
		l = append(l, temp)
	}
	return l
}

// eventsToSlice converts events from the format []trace.Event to []Event for exporting
func eventsToSlice(events []export.Event) []Event {
	var e []Event
	for _, v := range events {
		temp := Event{
			Ts:    v.Time.UnixNano(),
			Name:  v.Name,
			Attrs: attributesToMap(v.Attributes),
		}
		e = append(e, temp)
	}
	return e
}
