package jaeger_test

import (
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/propagation"
)

func ExampleJaeger() {
	jaeger := jaeger.Jaeger{}
	// register jaeger propagator
	global.SetPropagators(propagation.New(
		propagation.WithExtractors(jaeger),
		propagation.WithInjectors(jaeger),
	))
}
