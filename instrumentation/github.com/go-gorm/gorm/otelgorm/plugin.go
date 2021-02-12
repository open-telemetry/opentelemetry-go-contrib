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

package otelgorm

import (
	"fmt"

	"gorm.io/gorm"

	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	defaultTracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/go-gorm/gorm/otelgorm"

	callBackBeforeName = "otel:before"
	callBackAfterName  = "otel:after"

	opCreate = "INSERT"
	opQuery  = "SELECT"
	opDelete = "DELETE"
	opUpdate = "UPDATE"
)

type gormHookFunc func(tx *gorm.DB)

type OtelPlugin struct {
	cfg    *config
	tracer oteltrace.Tracer
}

func (op *OtelPlugin) Name() string {
	return "OpenTelemetryPlugin"
}

// NewPlugin initialize a new gorm.DB plugin that traces queries
// You may pass optional Options to the function
func NewPlugin(opts ...Option) *OtelPlugin {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.tracerProvider == nil {
		cfg.tracerProvider = otel.GetTracerProvider()
	}

	return &OtelPlugin{
		cfg: cfg,
		tracer: cfg.tracerProvider.Tracer(
			defaultTracerName,
			oteltrace.WithInstrumentationVersion(contrib.SemVersion()),
		),
	}
}

type registerCallback interface {
	Register(name string, fn func(*gorm.DB)) error
}

func beforeName(name string) string {
	return callBackBeforeName + "_" + name
}

func afterName(name string) string {
	return callBackAfterName + "_" + name
}

func (op *OtelPlugin) Initialize(db *gorm.DB) error {
	registerHooks := []struct {
		callback registerCallback
		hook     gormHookFunc
		name     string
	}{
		// before hooks
		{db.Callback().Create().Before("gorm:before_create"), op.before(opCreate), beforeName("create")},
		{db.Callback().Query().Before("gorm:query"), op.before(opQuery), beforeName("query")},
		{db.Callback().Delete().Before("gorm:before_delete"), op.before(opDelete), beforeName("delete")},
		{db.Callback().Update().Before("gorm:before_update"), op.before(opUpdate), beforeName("update")},
		{db.Callback().Row().Before("gorm:row"), op.before(""), beforeName("row")},
		{db.Callback().Raw().Before("gorm:raw"), op.before(""), beforeName("raw")},

		// after hooks
		{db.Callback().Create().After("gorm:after_create"), op.after(opCreate), afterName("create")},
		{db.Callback().Query().After("gorm:after_query"), op.after(opQuery), afterName("select")},
		{db.Callback().Delete().After("gorm:after_delete"), op.after(opDelete), afterName("delete")},
		{db.Callback().Update().After("gorm:after_update"), op.after(opUpdate), afterName("update")},
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
