package otelgorm

import (
	"fmt"

	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const (
	defaultTracerName  = "go.opentelemetry.io/contrib/instrumentation/github.com/go-gorm/gorm/otelgorm"
	defaultServiceName = "gorm"

	callBackBeforeName = "otel:before"
	callBackAfterName  = "otel:after"
	spanName           = "gorm_sql_query"
)

type gormHookFunc func(tx *gorm.DB)

type otelPlugin struct{
	cfg *config
	tracer oteltrace.Tracer
}

func (op *otelPlugin) Name() string {
	return "OpenTelemetryPlugin"
}

func NewPlugin(opts ...Option) *otelPlugin {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.serviceName == "" {
		cfg.serviceName = defaultServiceName
	}

	if cfg.tracerProvider == nil {
		cfg.tracerProvider = otel.GetTracerProvider()
	}

	return &otelPlugin{
		cfg: cfg,
	}
}

type registerCallback interface {
	Register(name string, fn func(*gorm.DB)) error
}

func (op *otelPlugin) Initialize(db *gorm.DB) error {
	registerHooks := []struct {
		callback registerCallback
		hook     gormHookFunc
		name     string
	}{
		// before hooks
		{db.Callback().Create().Before("gorm:before_create"), op.before, beforeName("create")},
		{db.Callback().Query().Before("gorm:query"), op.before, beforeName("query")},
		{db.Callback().Delete().Before("gorm:before_delete"), op.before, beforeName("delete")},
		{db.Callback().Update().Before("gorm:before_update"), op.before, beforeName("update")},
		{db.Callback().Row().Before("gorm:row"), op.before, beforeName("row")},
		{db.Callback().Raw().Before("gorm:raw"), op.before, beforeName("raw")},

		// after hooks
		{db.Callback().Create().After("gorm:after_create"), op.after("INSERT"), afterName("create")},
		{db.Callback().Query().After("gorm:after_query"), op.after("SELECT"), afterName("select")},
		{db.Callback().Delete().After("gorm:after_delete"), op.after("DELETE"), afterName("delete")},
		{db.Callback().Update().After("gorm:after_update"), op.after("UPDATE"), afterName("update")},
		{db.Callback().Row().After("gorm:row"), op.after(""), afterName("row")},
		{db.Callback().Raw().After("gorm:raw"), op.after(""), afterName("raw")},
	}

	for _, h := range registerHooks {
		if err := h.callback.Register(h.name, h.hook); err != nil {
			return fmt.Errorf("register %s hook: %w", h.name, err)
		}
	}

	return nil
}
