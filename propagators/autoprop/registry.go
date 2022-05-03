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

package autoprop

import (
	"sync"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel/propagation"
)

// none is the specical "propagator" name that means no propagator shall be
// configured.
const none = "none"

// envRegistry is the index of all supported environment variable
// values and their mapping to a TextMapPropagator.
var envRegistry = &registry{
	index: map[string]propagation.TextMapPropagator{
		// W3C Trace Context.
		"tracecontext": propagation.TraceContext{},
		// W3C Baggage.
		"baggage": propagation.Baggage{},
		// B3 single-header format.
		"b3": b3.New(),
		// B3 multi-header format.
		"b3multi": b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
		// Jaeger.
		"jaeger": jaeger.Jaeger{},
		// AWS X-Ray.
		"xray": xray.Propagator{},
		// OpenTracing Trace.
		"ottrace": ot.OT{},

		// No-op TextMapPropagator.
		none: propagation.NewCompositeTextMapPropagator(),
	},
}

// registry maintains an index of propagator names to TextMapPropagator
// implementations that is safe for concurrent use by multiple goroutines
// without additional locking or coordination.
type registry struct {
	mu    sync.Mutex
	index map[string]propagation.TextMapPropagator
}

// load returns the value stored in the registry index for a key, or nil if no
// value is present. The ok result indicates whether value was found in the
// index.
func (r *registry) load(key string) (p propagation.TextMapPropagator, ok bool) {
	r.mu.Lock()
	p, ok = r.index[key]
	r.mu.Unlock()
	return p, ok
}

// store sets the value for a key.
func (r *registry) store(key string, value propagation.TextMapPropagator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.index == nil {
		r.index = map[string]propagation.TextMapPropagator{key: value}
		return
	}
	r.index[key] = value
}

// RegisterTextMapPropagator sets the TextMapPropagator p to be used when the
// OTEL_PROPAGATORS environment variable contains the propagator name. This
// allows the default supported environment TextMapPropagators to be extended
// with 3rd-part implementations.
func RegisterTextMapPropagator(name string, p propagation.TextMapPropagator) {
	envRegistry.store(name, p)
}
