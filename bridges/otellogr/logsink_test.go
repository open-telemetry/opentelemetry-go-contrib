package otellogr

import (
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/log/noop"
)

func TestWIP(t *testing.T) {
	provider := noop.NewLoggerProvider()
	logger := logr.New(NewLogSink(WithLoggerProvider(provider)))

	// Tests are WIP, the following code is just to make sure the code compiles

	logger.Info("This is a test message")

	logger.Error(errors.New("This is a test error message"), "This is a test error message")

	logger.V(1).Info("This is a test message with verbosity level 1")

	logger.WithName("test").Info("This is a test message with a name")

	logger.WithValues("key", "value").Info("This is a test message with values")

	logger.WithName("test").WithValues("key", "value").Info("This is a test message with a name and values")

	logger.Info("This is a test message with a name and values", "key", "value")

	logger.Info("This is a test message with a name and values", "int", 10, "bool", true)
}
