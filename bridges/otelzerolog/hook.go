package otelzerolog

import (
	"github.com/rs/zerolog"
	"fmt"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type config struct {
	provider  log.LoggerProvider
	version   string
	schemaURL string

}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	return c
}

func (c config) logger(name string) log.Logger {
	var opts []log.LoggerOption
	if c.version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.version))
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}
	return c.provider.Logger(name, opts...)
}

// Option configures a Hook.
type Option interface {
	apply(config) config
}
type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [Hook]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [Hook]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [Hook].
//
// By default if this Option is not provided, the Hook will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}
func NewHook(name string, options ...Option) *SeverityHook {
	cfg := newConfig(options)
	return &SeverityHook{
		logger: cfg.logger(name),
	}
}
type SeverityHook struct{
	logger log.Logger
	levels zerolog.Level
}

// Levels returns the list of log levels we want to be sent to OpenTelemetry.
func (h *SeverityHook) Levels() zerolog.Level {
	return h.levels
}

func (h SeverityHook) Run(e *zerolog.Event, level zerolog.Level, msg string) error {
    if level != zerolog.NoLevel {
        e.Str("severity", level.String())
    }
	h.logger.Emit(e.GetCtx(),h.convertEvent(e,level,msg))
	return nil
}
func(h *SeverityHook) convertEvent(e *zerolog.Event,level zerolog.Level, msg string) log.Record{
	var record log.Record
	record.SetTimestamp(zerolog.TimestampFunc())
	record.SetBody(log.StringValue(msg))
	const sevOffset = zerolog.Level(log.SeverityDebug) - zerolog.DebugLevel
	record.SetSeverity(log.Severity(level + sevOffset))
	fields := extractFields(e)

	record.AddAttributes(convertFields(fields,msg)...);
	return record
}
func extractFields(_ *zerolog.Event) map[string]interface{} {
	// Here you would implement the logic to extract fields from the zerolog event
	// This might involve using reflection or zerolog internals if necessary
	fields := make(map[string]interface{})
	// Dummy implementation - replace with actual field extraction
	
	return fields
}
func convertFields(fields map[string]interface{}, msg string) []log.KeyValue {
	kvs := make([]log.KeyValue, 0, len(fields))
	kvs = append(kvs, log.String("message", msg))
	for k, v := range fields {
		kvs = append(kvs, convertAttribute(k, v))
	}
	return kvs
}

func convertAttribute(key string, value interface{}) log.KeyValue {
	switch v := value.(type) {
		case bool:
			return log.Bool(key, v)
		case []byte:
			return log.String(key, string(v))
		case float64:
			return log.Float64(key, v)
		case int:
			return log.Int(key, v)
		case int64:
			return log.Int64(key, v)
		case string:
			return log.String(key, v)
		default:
			// Fallback to string representation for unhandled types
			return log.String(key, fmt.Sprintf("%v", v))
	}
}




