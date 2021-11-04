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

package otelsql

import (
	"go.opentelemetry.io/otel/attribute"
)

const defaultInstanceName = "default"

// wrapper holds configuration of our sql tracing middleware.
// By default, all options are set to false intentionally when creating a wrapped
// driver and provide the most sensible default with both performance and
// security in mind.
//go:generate go-option -type=wrapper
type wrapper struct {
	// AllowRoot, if set to true, will allow otelsql to create root spans in
	// absence of existing spans or even context.
	// Default is to not trace otelsql calls if no existing parent span is found
	// in context or when using methods not taking context.
	AllowRoot bool

	// Ping, if set to true, will enable the creation of spans on Ping requests.
	Ping bool

	// RowsNext, if set to true, will enable the creation of spans on RowsNext
	// calls. This can result in many spans.
	RowsNext bool

	// RowsClose, if set to true, will enable the creation of spans on RowsClose
	// calls.
	RowsClose bool

	// RowsAffected, if set to true, will enable the creation of spans on
	// RowsAffected calls.
	RowsAffected bool

	// LastInsertID, if set to true, will enable the creation of spans on
	// LastInsertId calls.
	LastInsertID bool

	// Query, if set to true, will enable recording of sql queries in spans.
	// Only allow this if it is safe to have queries recorded with respect to
	// security.
	Query bool

	// QueryParams, if set to true, will enable recording of parameters used
	// with parametrized queries. Only allow this if it is safe to have
	// parameters recorded with respect to security.
	// This setting is a noop if the Query option is set to false.
	QueryParams bool

	// DefaultAttributes will be set to each span as default.
	DefaultAttributes []attribute.KeyValue

	// InstanceName identifies database.
	InstanceName string

	// DisableErrSkip, if set to true, will suppress driver.ErrSkip errors in spans.
	DisableErrSkip bool
}

func (w *wrapper) SetDefaults() {
	// https://opentracing.io/specification/conventions/
	// db.type	string	Database type.
	// For any SQL database, "sql". For others, the lower-case database category, e.g. "cassandra", "hbase", or "redis".
	w.DefaultAttributes = append(w.DefaultAttributes, attribute.String("db.type", "sql"))
}

// WithAllWrapperOptions enables all available trace options.
func WithAllWrapperOptions() WrapperOption {
	return WrapperOptionFunc(func(o *wrapper) {
		*o = AllWrapperOptions
	})
}

// AllWrapperOptions has all tracing options enabled.
var AllWrapperOptions = wrapper{
	AllowRoot:         true,
	Ping:              true,
	RowsNext:          true,
	RowsClose:         true,
	RowsAffected:      true,
	LastInsertID:      true,
	Query:             true,
	QueryParams:       true,
	DefaultAttributes: []attribute.KeyValue{attribute.String("db.type", "sql")},
}

// WithOptions sets our otelsql tracing middleware options through a single
// WrapperOptions object.
func WithOptions(options wrapper) WrapperOption {
	return WrapperOptionFunc(func(o *wrapper) {
		*o = options
		o.DefaultAttributes = append(
			[]attribute.KeyValue{attribute.String("db.type", "sql")}, options.DefaultAttributes...,
		)
	})
}
