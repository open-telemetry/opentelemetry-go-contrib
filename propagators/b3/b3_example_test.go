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

package b3_test

import (
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
)

func ExampleNew() {
	p := b3.New()
	// Register the B3 propagator globally.
	otel.SetTextMapPropagator(p)
}

func ExampleNew_injectEncoding() {
	// Create a B3 propagator configured to inject context with both multiple
	// and single header B3 HTTP encoding.
	p := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader | b3.B3SingleHeader))
	otel.SetTextMapPropagator(p)
}
