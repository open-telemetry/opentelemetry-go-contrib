// Code generated by github.com/atombender/go-jsonschema, DO NOT EDIT.

package config

import "encoding/json"
import "fmt"
import "reflect"

type AttributeLimits struct {
	// AttributeCountLimit corresponds to the JSON schema field
	// "attribute_count_limit".
	AttributeCountLimit *int `mapstructure:"attribute_count_limit,omitempty"`

	// AttributeValueLengthLimit corresponds to the JSON schema field
	// "attribute_value_length_limit".
	AttributeValueLengthLimit *int `mapstructure:"attribute_value_length_limit,omitempty"`

	AdditionalProperties interface{}
}

type Attributes struct {
	// ServiceName corresponds to the JSON schema field "service.name".
	ServiceName *string `mapstructure:"service.name,omitempty"`

	AdditionalProperties interface{}
}

type BatchLogRecordProcessor struct {
	// ExportTimeout corresponds to the JSON schema field "export_timeout".
	ExportTimeout *int `mapstructure:"export_timeout,omitempty"`

	// Exporter corresponds to the JSON schema field "exporter".
	Exporter LogRecordExporter `mapstructure:"exporter"`

	// MaxExportBatchSize corresponds to the JSON schema field
	// "max_export_batch_size".
	MaxExportBatchSize *int `mapstructure:"max_export_batch_size,omitempty"`

	// MaxQueueSize corresponds to the JSON schema field "max_queue_size".
	MaxQueueSize *int `mapstructure:"max_queue_size,omitempty"`

	// ScheduleDelay corresponds to the JSON schema field "schedule_delay".
	ScheduleDelay *int `mapstructure:"schedule_delay,omitempty"`
}

type BatchSpanProcessor struct {
	// ExportTimeout corresponds to the JSON schema field "export_timeout".
	ExportTimeout *int `mapstructure:"export_timeout,omitempty"`

	// Exporter corresponds to the JSON schema field "exporter".
	Exporter SpanExporter `mapstructure:"exporter"`

	// MaxExportBatchSize corresponds to the JSON schema field
	// "max_export_batch_size".
	MaxExportBatchSize *int `mapstructure:"max_export_batch_size,omitempty"`

	// MaxQueueSize corresponds to the JSON schema field "max_queue_size".
	MaxQueueSize *int `mapstructure:"max_queue_size,omitempty"`

	// ScheduleDelay corresponds to the JSON schema field "schedule_delay".
	ScheduleDelay *int `mapstructure:"schedule_delay,omitempty"`
}

type Common map[string]interface{}

type Console map[string]interface{}

type Headers map[string]string

type LogRecordExporter struct {
	// OTLP corresponds to the JSON schema field "otlp".
	OTLP *OTLP `mapstructure:"otlp,omitempty"`

	AdditionalProperties interface{}
}

type LogRecordLimits struct {
	// AttributeCountLimit corresponds to the JSON schema field
	// "attribute_count_limit".
	AttributeCountLimit *int `mapstructure:"attribute_count_limit,omitempty"`

	// AttributeValueLengthLimit corresponds to the JSON schema field
	// "attribute_value_length_limit".
	AttributeValueLengthLimit *int `mapstructure:"attribute_value_length_limit,omitempty"`
}

type LogRecordProcessor struct {
	// Batch corresponds to the JSON schema field "batch".
	Batch *BatchLogRecordProcessor `mapstructure:"batch,omitempty"`

	// Simple corresponds to the JSON schema field "simple".
	Simple *SimpleLogRecordProcessor `mapstructure:"simple,omitempty"`

	AdditionalProperties interface{}
}

type LoggerProvider struct {
	// Limits corresponds to the JSON schema field "limits".
	Limits *LogRecordLimits `mapstructure:"limits,omitempty"`

	// Processors corresponds to the JSON schema field "processors".
	Processors []LogRecordProcessor `mapstructure:"processors,omitempty"`
}

type MeterProvider struct {
	// Readers corresponds to the JSON schema field "readers".
	Readers []MetricReader `mapstructure:"readers,omitempty"`

	// Views corresponds to the JSON schema field "views".
	Views []View `mapstructure:"views,omitempty"`
}

type MetricExporter struct {
	// Console corresponds to the JSON schema field "console".
	Console Console `mapstructure:"console,omitempty"`

	// OTLP corresponds to the JSON schema field "otlp".
	OTLP *OTLPMetric `mapstructure:"otlp,omitempty"`

	// Prometheus corresponds to the JSON schema field "prometheus".
	Prometheus *Prometheus `mapstructure:"prometheus,omitempty"`

	AdditionalProperties interface{}
}

type MetricReader struct {
	// Periodic corresponds to the JSON schema field "periodic".
	Periodic *PeriodicMetricReader `mapstructure:"periodic,omitempty"`

	// Pull corresponds to the JSON schema field "pull".
	Pull *PullMetricReader `mapstructure:"pull,omitempty"`
}

type OTLP struct {
	// Certificate corresponds to the JSON schema field "certificate".
	Certificate *string `mapstructure:"certificate,omitempty"`

	// ClientCertificate corresponds to the JSON schema field "client_certificate".
	ClientCertificate *string `mapstructure:"client_certificate,omitempty"`

	// ClientKey corresponds to the JSON schema field "client_key".
	ClientKey *string `mapstructure:"client_key,omitempty"`

	// Compression corresponds to the JSON schema field "compression".
	Compression *string `mapstructure:"compression,omitempty"`

	// Endpoint corresponds to the JSON schema field "endpoint".
	Endpoint string `mapstructure:"endpoint"`

	// Headers corresponds to the JSON schema field "headers".
	Headers Headers `mapstructure:"headers,omitempty"`

	// Protocol corresponds to the JSON schema field "protocol".
	Protocol string `mapstructure:"protocol"`

	// Timeout corresponds to the JSON schema field "timeout".
	Timeout *int `mapstructure:"timeout,omitempty"`
}

type OTLPMetric struct {
	// Certificate corresponds to the JSON schema field "certificate".
	Certificate *string `mapstructure:"certificate,omitempty"`

	// ClientCertificate corresponds to the JSON schema field "client_certificate".
	ClientCertificate *string `mapstructure:"client_certificate,omitempty"`

	// ClientKey corresponds to the JSON schema field "client_key".
	ClientKey *string `mapstructure:"client_key,omitempty"`

	// Compression corresponds to the JSON schema field "compression".
	Compression *string `mapstructure:"compression,omitempty"`

	// DefaultHistogramAggregation corresponds to the JSON schema field
	// "default_histogram_aggregation".
	DefaultHistogramAggregation *OTLPMetricDefaultHistogramAggregation `mapstructure:"default_histogram_aggregation,omitempty"`

	// Endpoint corresponds to the JSON schema field "endpoint".
	Endpoint string `mapstructure:"endpoint"`

	// Headers corresponds to the JSON schema field "headers".
	Headers Headers `mapstructure:"headers,omitempty"`

	// Protocol corresponds to the JSON schema field "protocol".
	Protocol string `mapstructure:"protocol"`

	// TemporalityPreference corresponds to the JSON schema field
	// "temporality_preference".
	TemporalityPreference *string `mapstructure:"temporality_preference,omitempty"`

	// Timeout corresponds to the JSON schema field "timeout".
	Timeout *int `mapstructure:"timeout,omitempty"`
}

type OTLPMetricDefaultHistogramAggregation string

const OTLPMetricDefaultHistogramAggregationBase2ExponentialBucketHistogram OTLPMetricDefaultHistogramAggregation = "base2_exponential_bucket_histogram"
const OTLPMetricDefaultHistogramAggregationExplicitBucketHistogram OTLPMetricDefaultHistogramAggregation = "explicit_bucket_histogram"

type OpenTelemetryConfiguration struct {
	// AttributeLimits corresponds to the JSON schema field "attribute_limits".
	AttributeLimits *AttributeLimits `mapstructure:"attribute_limits,omitempty"`

	// Disabled corresponds to the JSON schema field "disabled".
	Disabled *bool `mapstructure:"disabled,omitempty"`

	// FileFormat corresponds to the JSON schema field "file_format".
	FileFormat string `mapstructure:"file_format"`

	// LoggerProvider corresponds to the JSON schema field "logger_provider".
	LoggerProvider *LoggerProvider `mapstructure:"logger_provider,omitempty"`

	// MeterProvider corresponds to the JSON schema field "meter_provider".
	MeterProvider *MeterProvider `mapstructure:"meter_provider,omitempty"`

	// Propagator corresponds to the JSON schema field "propagator".
	Propagator *Propagator `mapstructure:"propagator,omitempty"`

	// Resource corresponds to the JSON schema field "resource".
	Resource *Resource `mapstructure:"resource,omitempty"`

	// TracerProvider corresponds to the JSON schema field "tracer_provider".
	TracerProvider *TracerProvider `mapstructure:"tracer_provider,omitempty"`

	AdditionalProperties interface{}
}

type PeriodicMetricReader struct {
	// Exporter corresponds to the JSON schema field "exporter".
	Exporter MetricExporter `mapstructure:"exporter"`

	// Interval corresponds to the JSON schema field "interval".
	Interval *int `mapstructure:"interval,omitempty"`

	// Timeout corresponds to the JSON schema field "timeout".
	Timeout *int `mapstructure:"timeout,omitempty"`
}

type Prometheus struct {
	// Host corresponds to the JSON schema field "host".
	Host *string `mapstructure:"host,omitempty"`

	// Port corresponds to the JSON schema field "port".
	Port *int `mapstructure:"port,omitempty"`

	// WithoutScopeInfo corresponds to the JSON schema field "without_scope_info".
	WithoutScopeInfo *bool `mapstructure:"without_scope_info,omitempty"`

	// WithoutTypeSuffix corresponds to the JSON schema field "without_type_suffix".
	WithoutTypeSuffix *bool `mapstructure:"without_type_suffix,omitempty"`

	// WithoutUnits corresponds to the JSON schema field "without_units".
	WithoutUnits *bool `mapstructure:"without_units,omitempty"`
}

type Propagator struct {
	// Composite corresponds to the JSON schema field "composite".
	Composite []string `mapstructure:"composite,omitempty"`

	AdditionalProperties interface{}
}

type PullMetricReader struct {
	// Exporter corresponds to the JSON schema field "exporter".
	Exporter MetricExporter `mapstructure:"exporter"`
}

type Resource struct {
	// Attributes corresponds to the JSON schema field "attributes".
	Attributes *Attributes `mapstructure:"attributes,omitempty"`

	// SchemaUrl corresponds to the JSON schema field "schema_url".
	SchemaUrl *string `mapstructure:"schema_url,omitempty"`
}

type Sampler struct {
	// AlwaysOff corresponds to the JSON schema field "always_off".
	AlwaysOff SamplerAlwaysOff `mapstructure:"always_off,omitempty"`

	// AlwaysOn corresponds to the JSON schema field "always_on".
	AlwaysOn SamplerAlwaysOn `mapstructure:"always_on,omitempty"`

	// JaegerRemote corresponds to the JSON schema field "jaeger_remote".
	JaegerRemote *SamplerJaegerRemote `mapstructure:"jaeger_remote,omitempty"`

	// ParentBased corresponds to the JSON schema field "parent_based".
	ParentBased *SamplerParentBased `mapstructure:"parent_based,omitempty"`

	// TraceIDRatioBased corresponds to the JSON schema field "trace_id_ratio_based".
	TraceIDRatioBased *SamplerTraceIDRatioBased `mapstructure:"trace_id_ratio_based,omitempty"`

	AdditionalProperties interface{}
}

type SamplerAlwaysOff map[string]interface{}

type SamplerAlwaysOn map[string]interface{}

type SamplerJaegerRemote struct {
	// Endpoint corresponds to the JSON schema field "endpoint".
	Endpoint *string `mapstructure:"endpoint,omitempty"`

	// InitialSampler corresponds to the JSON schema field "initial_sampler".
	InitialSampler *Sampler `mapstructure:"initial_sampler,omitempty"`

	// Interval corresponds to the JSON schema field "interval".
	Interval *int `mapstructure:"interval,omitempty"`
}

type SamplerParentBased struct {
	// LocalParentNotSampled corresponds to the JSON schema field
	// "local_parent_not_sampled".
	LocalParentNotSampled *Sampler `mapstructure:"local_parent_not_sampled,omitempty"`

	// LocalParentSampled corresponds to the JSON schema field "local_parent_sampled".
	LocalParentSampled *Sampler `mapstructure:"local_parent_sampled,omitempty"`

	// RemoteParentNotSampled corresponds to the JSON schema field
	// "remote_parent_not_sampled".
	RemoteParentNotSampled *Sampler `mapstructure:"remote_parent_not_sampled,omitempty"`

	// RemoteParentSampled corresponds to the JSON schema field
	// "remote_parent_sampled".
	RemoteParentSampled *Sampler `mapstructure:"remote_parent_sampled,omitempty"`

	// Root corresponds to the JSON schema field "root".
	Root *Sampler `mapstructure:"root,omitempty"`
}

type SamplerTraceIDRatioBased struct {
	// Ratio corresponds to the JSON schema field "ratio".
	Ratio *float64 `mapstructure:"ratio,omitempty"`
}

type SimpleLogRecordProcessor struct {
	// Exporter corresponds to the JSON schema field "exporter".
	Exporter LogRecordExporter `mapstructure:"exporter"`
}

type SimpleSpanProcessor struct {
	// Exporter corresponds to the JSON schema field "exporter".
	Exporter SpanExporter `mapstructure:"exporter"`
}

type SpanExporter struct {
	// Console corresponds to the JSON schema field "console".
	Console Console `mapstructure:"console,omitempty"`

	// OTLP corresponds to the JSON schema field "otlp".
	OTLP *OTLP `mapstructure:"otlp,omitempty"`

	// Zipkin corresponds to the JSON schema field "zipkin".
	Zipkin *Zipkin `mapstructure:"zipkin,omitempty"`

	AdditionalProperties interface{}
}

type SpanLimits struct {
	// AttributeCountLimit corresponds to the JSON schema field
	// "attribute_count_limit".
	AttributeCountLimit *int `mapstructure:"attribute_count_limit,omitempty"`

	// AttributeValueLengthLimit corresponds to the JSON schema field
	// "attribute_value_length_limit".
	AttributeValueLengthLimit *int `mapstructure:"attribute_value_length_limit,omitempty"`

	// EventAttributeCountLimit corresponds to the JSON schema field
	// "event_attribute_count_limit".
	EventAttributeCountLimit *int `mapstructure:"event_attribute_count_limit,omitempty"`

	// EventCountLimit corresponds to the JSON schema field "event_count_limit".
	EventCountLimit *int `mapstructure:"event_count_limit,omitempty"`

	// LinkAttributeCountLimit corresponds to the JSON schema field
	// "link_attribute_count_limit".
	LinkAttributeCountLimit *int `mapstructure:"link_attribute_count_limit,omitempty"`

	// LinkCountLimit corresponds to the JSON schema field "link_count_limit".
	LinkCountLimit *int `mapstructure:"link_count_limit,omitempty"`
}

type SpanProcessor struct {
	// Batch corresponds to the JSON schema field "batch".
	Batch *BatchSpanProcessor `mapstructure:"batch,omitempty"`

	// Simple corresponds to the JSON schema field "simple".
	Simple *SimpleSpanProcessor `mapstructure:"simple,omitempty"`

	AdditionalProperties interface{}
}

type TracerProvider struct {
	// Limits corresponds to the JSON schema field "limits".
	Limits *SpanLimits `mapstructure:"limits,omitempty"`

	// Processors corresponds to the JSON schema field "processors".
	Processors []SpanProcessor `mapstructure:"processors,omitempty"`

	// Sampler corresponds to the JSON schema field "sampler".
	Sampler *Sampler `mapstructure:"sampler,omitempty"`
}

type View struct {
	// Selector corresponds to the JSON schema field "selector".
	Selector *ViewSelector `mapstructure:"selector,omitempty"`

	// Stream corresponds to the JSON schema field "stream".
	Stream *ViewStream `mapstructure:"stream,omitempty"`
}

type ViewSelector struct {
	// InstrumentName corresponds to the JSON schema field "instrument_name".
	InstrumentName *string `mapstructure:"instrument_name,omitempty"`

	// InstrumentType corresponds to the JSON schema field "instrument_type".
	InstrumentType *ViewSelectorInstrumentType `mapstructure:"instrument_type,omitempty"`

	// MeterName corresponds to the JSON schema field "meter_name".
	MeterName *string `mapstructure:"meter_name,omitempty"`

	// MeterSchemaUrl corresponds to the JSON schema field "meter_schema_url".
	MeterSchemaUrl *string `mapstructure:"meter_schema_url,omitempty"`

	// MeterVersion corresponds to the JSON schema field "meter_version".
	MeterVersion *string `mapstructure:"meter_version,omitempty"`

	// Unit corresponds to the JSON schema field "unit".
	Unit *string `mapstructure:"unit,omitempty"`
}

type ViewSelectorInstrumentType string

const ViewSelectorInstrumentTypeCounter ViewSelectorInstrumentType = "counter"
const ViewSelectorInstrumentTypeHistogram ViewSelectorInstrumentType = "histogram"
const ViewSelectorInstrumentTypeObservableCounter ViewSelectorInstrumentType = "observable_counter"
const ViewSelectorInstrumentTypeObservableGauge ViewSelectorInstrumentType = "observable_gauge"
const ViewSelectorInstrumentTypeObservableUpDownCounter ViewSelectorInstrumentType = "observable_up_down_counter"
const ViewSelectorInstrumentTypeUpDownCounter ViewSelectorInstrumentType = "up_down_counter"

type ViewStream struct {
	// Aggregation corresponds to the JSON schema field "aggregation".
	Aggregation *ViewStreamAggregation `mapstructure:"aggregation,omitempty"`

	// AttributeKeys corresponds to the JSON schema field "attribute_keys".
	AttributeKeys []string `mapstructure:"attribute_keys,omitempty"`

	// Description corresponds to the JSON schema field "description".
	Description *string `mapstructure:"description,omitempty"`

	// Name corresponds to the JSON schema field "name".
	Name *string `mapstructure:"name,omitempty"`
}

type ViewStreamAggregation struct {
	// Base2ExponentialBucketHistogram corresponds to the JSON schema field
	// "base2_exponential_bucket_histogram".
	Base2ExponentialBucketHistogram *ViewStreamAggregationBase2ExponentialBucketHistogram `mapstructure:"base2_exponential_bucket_histogram,omitempty"`

	// Default corresponds to the JSON schema field "default".
	Default ViewStreamAggregationDefault `mapstructure:"default,omitempty"`

	// Drop corresponds to the JSON schema field "drop".
	Drop ViewStreamAggregationDrop `mapstructure:"drop,omitempty"`

	// ExplicitBucketHistogram corresponds to the JSON schema field
	// "explicit_bucket_histogram".
	ExplicitBucketHistogram *ViewStreamAggregationExplicitBucketHistogram `mapstructure:"explicit_bucket_histogram,omitempty"`

	// LastValue corresponds to the JSON schema field "last_value".
	LastValue ViewStreamAggregationLastValue `mapstructure:"last_value,omitempty"`

	// Sum corresponds to the JSON schema field "sum".
	Sum ViewStreamAggregationSum `mapstructure:"sum,omitempty"`
}

type ViewStreamAggregationBase2ExponentialBucketHistogram struct {
	// MaxScale corresponds to the JSON schema field "max_scale".
	MaxScale *int `mapstructure:"max_scale,omitempty"`

	// MaxSize corresponds to the JSON schema field "max_size".
	MaxSize *int `mapstructure:"max_size,omitempty"`

	// RecordMinMax corresponds to the JSON schema field "record_min_max".
	RecordMinMax *bool `mapstructure:"record_min_max,omitempty"`
}

type ViewStreamAggregationDefault map[string]interface{}

type ViewStreamAggregationDrop map[string]interface{}

type ViewStreamAggregationExplicitBucketHistogram struct {
	// Boundaries corresponds to the JSON schema field "boundaries".
	Boundaries []float64 `mapstructure:"boundaries,omitempty"`

	// RecordMinMax corresponds to the JSON schema field "record_min_max".
	RecordMinMax *bool `mapstructure:"record_min_max,omitempty"`
}

type ViewStreamAggregationLastValue map[string]interface{}

type ViewStreamAggregationSum map[string]interface{}

type Zipkin struct {
	// Endpoint corresponds to the JSON schema field "endpoint".
	Endpoint string `mapstructure:"endpoint"`

	// Timeout corresponds to the JSON schema field "timeout".
	Timeout *int `mapstructure:"timeout,omitempty"`
}

var enumValues_OTLPMetricDefaultHistogramAggregation = []interface{}{
	"explicit_bucket_histogram",
	"base2_exponential_bucket_histogram",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPMetricDefaultHistogramAggregation) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValues_OTLPMetricDefaultHistogramAggregation {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValues_OTLPMetricDefaultHistogramAggregation, v)
	}
	*j = OTLPMetricDefaultHistogramAggregation(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Attributes) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain Attributes
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = Attributes(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Zipkin) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["endpoint"]; !ok || v == nil {
		return fmt.Errorf("field endpoint in Zipkin: required")
	}
	type Plain Zipkin
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Zipkin(plain)
	return nil
}

var enumValues_ViewSelectorInstrumentType = []interface{}{
	"counter",
	"histogram",
	"observable_counter",
	"observable_gauge",
	"observable_up_down_counter",
	"up_down_counter",
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SpanExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain SpanExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = SpanExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PullMetricReader) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["exporter"]; !ok || v == nil {
		return fmt.Errorf("field exporter in PullMetricReader: required")
	}
	type Plain PullMetricReader
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PullMetricReader(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchSpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["exporter"]; !ok || v == nil {
		return fmt.Errorf("field exporter in BatchSpanProcessor: required")
	}
	type Plain BatchSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = BatchSpanProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PeriodicMetricReader) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["exporter"]; !ok || v == nil {
		return fmt.Errorf("field exporter in PeriodicMetricReader: required")
	}
	type Plain PeriodicMetricReader
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PeriodicMetricReader(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *MetricExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain MetricExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = MetricExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLPMetric) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["endpoint"]; !ok || v == nil {
		return fmt.Errorf("field endpoint in OTLPMetric: required")
	}
	if v, ok := raw["protocol"]; !ok || v == nil {
		return fmt.Errorf("field protocol in OTLPMetric: required")
	}
	type Plain OTLPMetric
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = OTLPMetric(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ViewSelectorInstrumentType) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValues_ViewSelectorInstrumentType {
		if reflect.DeepEqual(v, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValues_ViewSelectorInstrumentType, v)
	}
	*j = ViewSelectorInstrumentType(v)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Propagator) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain Propagator
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = Propagator(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *LogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain LogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = LogRecordProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Sampler) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain Sampler
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = Sampler(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["exporter"]; !ok || v == nil {
		return fmt.Errorf("field exporter in SimpleLogRecordProcessor: required")
	}
	type Plain SimpleLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SimpleLogRecordProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SimpleSpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["exporter"]; !ok || v == nil {
		return fmt.Errorf("field exporter in SimpleSpanProcessor: required")
	}
	type Plain SimpleSpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SimpleSpanProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BatchLogRecordProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["exporter"]; !ok || v == nil {
		return fmt.Errorf("field exporter in BatchLogRecordProcessor: required")
	}
	type Plain BatchLogRecordProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = BatchLogRecordProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *LogRecordExporter) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain LogRecordExporter
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = LogRecordExporter(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SpanProcessor) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain SpanProcessor
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = SpanProcessor(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OTLP) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["endpoint"]; !ok || v == nil {
		return fmt.Errorf("field endpoint in OTLP: required")
	}
	if v, ok := raw["protocol"]; !ok || v == nil {
		return fmt.Errorf("field protocol in OTLP: required")
	}
	type Plain OTLP
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = OTLP(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *AttributeLimits) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type Plain AttributeLimits
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = AttributeLimits(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OpenTelemetryConfiguration) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["file_format"]; !ok || v == nil {
		return fmt.Errorf("field file_format in OpenTelemetryConfiguration: required")
	}
	type Plain OpenTelemetryConfiguration
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if v, ok := raw[""]; !ok || v == nil {
		plain.AdditionalProperties = map[string]interface{}{}
	}
	*j = OpenTelemetryConfiguration(plain)
	return nil
}
