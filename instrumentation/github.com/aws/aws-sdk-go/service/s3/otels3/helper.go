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

package otels3

import (
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/trace"
)

// appendSpanAndTraceIDFromSpan extracts the trace id and span id from a span using the context field.
// It returns a list of attributes with the span id and trace id appended.
func appendSpanAndTraceIDFromSpan(attrs []label.KeyValue, span trace.Span) []label.KeyValue {
	return append(attrs,
		label.String("event.spanId", span.SpanContext().SpanID.String()),
		label.String("event.traceId", span.SpanContext().TraceID.String()),
	)
}
