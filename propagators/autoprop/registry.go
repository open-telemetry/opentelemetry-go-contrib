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

package autoprop // import "go.opentelemetry.io/contrib/propagators/autoprop"

import (
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel/propagation"
)

// none is the special "propagator" name that means no propagator shall be
// configured.
const none = "none"

// envRegistry is the index of all supported environment variable
// values and their mapping to a TextMapPropagator.
var envRegistry = &registry{
	names: map[string]propagation.TextMapPropagator{
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

// registry maintains a map of propagator names to TextMapPropagator
// implementations that is safe for concurrent use by multiple goroutines
// without additional locking or coordination.
type registry struct {
	mu    sync.Mutex
	names map[string]propagation.TextMapPropagator
}

// load returns the value stored in the registry index for a key, or nil if no
// value is present. The ok result indicates whether value was found in the
// index.
func (r *registry) load(key string) (p propagation.TextMapPropagator, ok bool) {
	r.mu.Lock()
	p, ok = r.names[key]
	r.mu.Unlock()
	return p, ok
}

var errDupReg = errors.New("duplicate registration")

// store sets the value for a key if is not already in the registry. errDupReg
// is returned if the registry already contains key.
func (r *registry) store(key string, value propagation.TextMapPropagator) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.names == nil {
		r.names = map[string]propagation.TextMapPropagator{key: value}
		return nil
	}
	if _, ok := r.names[key]; ok {
		return fmt.Errorf("%w: %q", errDupReg, key)
	}
	r.names[key] = value
	return nil
}

// drop removes key from the registry if it exists, otherwise nothing.
func (r *registry) drop(key string) {
	r.mu.Lock()
	delete(r.names, key)
	r.mu.Unlock()
}

// RegisterTextMapPropagator sets the TextMapPropagator p to be used when the
// OTEL_PROPAGATORS environment variable contains the propagator name. This
// will panic if name has already been registered or is a default
// (tracecontext, baggage, b3, b3multi, jaeger, xray, or ottrace).
func RegisterTextMapPropagator(name string, p propagation.TextMapPropagator) {
	if err := envRegistry.store(name, p); err != nil {
		// envRegistry.store will return errDupReg if name is already
		// registered. Panic here so the user is made aware of the duplicate
		// registration, which could be done by malicious code trying to
		// intercept cross-cutting concerns.
		//
		// Panic for all other errors as well. At this point there should not
		// be any other errors returned from the store operation. If there
		// are, alert the developer that adding them as soon as possible that
		// they need to be handled here.
		panic(err)
	}
}
