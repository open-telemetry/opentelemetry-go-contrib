package zap

import (
	"context"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.uber.org/zap/zapcore"
)

const (
	bridgeName = "go.opentelemetry.io/contrib/bridge/zapcore"
)

type config struct {
	scope instrumentation.Scope
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	var emptyScope instrumentation.Scope
	if c.scope == emptyScope {
		c.scope = instrumentation.Scope{
			Name:    bridgeName,
			Version: Version(),
		}
	}
	return c
}

func (c config) loggerArgs() (string, []log.LoggerOption) {
	var opts []log.LoggerOption
	if c.scope.Version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.scope.Version))
	}
	if c.scope.SchemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.scope.SchemaURL))
	}
	return c.scope.Name, opts
}

// Option configures a [Zapcore].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithInstrumentationScope returns an option that configures the scope of the
// [log.Logger] used by  zapcore
//
// By default if this Option is not provided, zapcore will use a default
// instrumentation scope describing this bridge package. It is recommended to
// provide this so log data can be associated with its source package or
// module.
func WithInstrumentationScope(scope instrumentation.Scope) Option {
	return optFunc(func(c config) config {
		c.scope = scope
		return c
	})
}

type OtelZapCore struct {
	logger log.Logger
	attr   []log.KeyValue
}

var (
	_ zapcore.Core = (*OtelZapCore)(nil)
)

// this function creates a new zapcore.Core that can be used with zap.New()
// this instance will translate zap logs to opentelemetry logs and export them
func NewOtelZapCore(lp log.LoggerProvider, opts ...Option) zapcore.Core {
	if lp == nil {
		// Do not panic.
		lp = noop.NewLoggerProvider()
	}

	name, loggerOpts := newConfig(opts).loggerArgs()
	// these options
	return &OtelZapCore{
		logger: lp.Logger(name,
			loggerOpts...,
		),
	}
}

func (o *OtelZapCore) Enabled(level zapcore.Level) bool {
	r := log.Record{}
	r.SetSeverity(getOtelLevel(level))

	// check how to propogate context
	ctx := context.Background()
	return o.logger.Enabled(ctx, r)

}

// return new zapcore with provided attr
func (o *OtelZapCore) With(fields []zapcore.Field) zapcore.Core {
	clone := o.clone()
	clone.attr = append(clone.attr, getAttr(fields)...)
	return clone
}

// TODO
func (o *OtelZapCore) Sync() error {
	return nil
}

func (o *OtelZapCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if o.Enabled(ent.Level) {
		return ce.AddCore(ent, o)
	}
	return ce
}

func (o *OtelZapCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	// we create record here to avoid heap allocation
	r := log.Record{}
	r.SetTimestamp(ent.Time)
	r.SetBody(log.StringValue(ent.Message))
	r.SetSeverity(getOtelLevel(ent.Level))

	// get attr from fields
	attr := getAttr(fields)
	// append attributes received from from parent logger
	addattr := append(attr, o.attr...)

	if len(addattr) > 0 {
		r.AddAttributes(addattr...)
	}

	// need to check how to propogate context here
	ctx := context.Background()
	o.logger.Emit(ctx, r)
	return nil
}

func (o *OtelZapCore) clone() *OtelZapCore {
	return &OtelZapCore{
		logger: o.logger,
		attr:   o.attr,
	}
}

func getAttr(fields []zapcore.Field) []log.KeyValue {
	enc := NewOtelObjectEncoder(len(fields))
	for i := range fields {
		fields[i].AddTo(enc)
	}
	return enc.cur
}

func getOtelLevel(level zapcore.Level) log.Severity {
	// should confirm this
	// the logic here is that
	// zapcore.Debug = -1 & logger.Debug = 3
	// zapcore.Info = 0   & logger.Info = 7 and so on
	sevOffset := 4*(level+2) + 1
	return log.Severity(level + sevOffset)
}
