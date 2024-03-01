// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autoprop_test

import (
	"fmt"
	"os"
	"sort"

	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func ExampleNewTextMapPropagator() {
	// NewTextMapPropagator returns a TraceContext and Baggage propagator by
	// default. The response of this function can be directly registered with
	// the go.opentelemetry.io/otel package.
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	fields := otel.GetTextMapPropagator().Fields()
	sort.Strings(fields)
	fmt.Println(fields)
	// Output: [baggage traceparent tracestate]
}

func ExampleNewTextMapPropagator_arguments() {
	// NewTextMapPropagator behaves the same as the
	// NewCompositeTextMapPropagator function in the
	// go.opentelemetry.io/otel/propagation package when TextMapPropagator are
	// passed as arguments.
	fields := autoprop.NewTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
		b3.New(),
	).Fields()
	sort.Strings(fields)
	fmt.Println(fields)
	// Output: [baggage traceparent tracestate x-b3-flags x-b3-sampled x-b3-spanid x-b3-traceid]
}

func ExampleNewTextMapPropagator_environment() {
	// Propagators set for the OTEL_PROPAGATORS environment variable take
	// precedence and will override any arguments passed to
	// NewTextMapPropagator.
	_ = os.Setenv("OTEL_PROPAGATORS", "b3,baggage")

	// Returns only a B3 and Baggage TextMapPropagator (i.e. does not include
	// TraceContext).
	fields := autoprop.NewTextMapPropagator(propagation.TraceContext{}).Fields()
	sort.Strings(fields)
	fmt.Println(fields)
	// Output: [baggage x-b3-flags x-b3-sampled x-b3-spanid x-b3-traceid]
}

type myTextMapPropagator struct{ propagation.TextMapPropagator }

func (myTextMapPropagator) Fields() []string {
	return []string{"my-header-val"}
}

func ExampleRegisterTextMapPropagator() {
	// To use your own or a 3rd-party exporter via the OTEL_PROPAGATORS
	// environment variable, it needs to be registered prior to calling
	// NewTextMapPropagator.
	autoprop.RegisterTextMapPropagator("custom-prop", myTextMapPropagator{})

	_ = os.Setenv("OTEL_PROPAGATORS", "custom-prop")
	fmt.Println(autoprop.NewTextMapPropagator().Fields())
	// Output: [my-header-val]
}

func ExampleGetTextMapPropagator() {
	prop, err := autoprop.TextMapPropagator("b3", "baggage")
	if err != nil {
		// Handle error appropriately.
		panic(err)
	}

	fields := prop.Fields()
	sort.Strings(fields)
	fmt.Println(fields)
	// Output: [baggage x-b3-flags x-b3-sampled x-b3-spanid x-b3-traceid]
}

func ExampleGetTextMapPropagator_custom() {
	// To use your own or a 3rd-party exporter it needs to be registered prior
	// to calling GetTextMapPropagator.
	autoprop.RegisterTextMapPropagator("custom-get-prop", myTextMapPropagator{})

	prop, err := autoprop.TextMapPropagator("custom-get-prop")
	if err != nil {
		// Handle error appropriately.
		panic(err)
	}

	fmt.Println(prop.Fields())
	// Output: [my-header-val]
}
