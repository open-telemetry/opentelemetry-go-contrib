// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envcar // import "go.opentelemetry.io/contrib/propagators/envcar"

import (
	"os"
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

// Carrier is a TextMapCarrier that uses environment variables as a storage
// medium for propagated key-value pairs. Keys passed to [Carrier.Get] and
// [Carrier.Set] are normalized before lookup or write. [Carrier.Keys] only
// lists environment variable names that are already normalized; on platforms
// with case-insensitive environment variable lookup (e.g. Windows) [Carrier.Get] may still match non-normalized names.
// This is useful for propagating values that are set in the environment
// and need to be accessed by different processes or services.
// The keys are uppercased to avoid case sensitivity issues across different
// operating systems and environments.
//
// If you do not set SetEnvFunc, [Carrier.Set] will do nothing.
// Using [os.Setenv] here is discouraged as the environment should
// be immutable:
// https://opentelemetry.io/docs/specs/otel/context/env-carriers/#environment-variable-immutability
type Carrier struct {
	// SetEnvFunc is the function that sets the environment variable.
	// [Carrier.Set] calls SetEnvFunc with a normalized key.
	// Usually, you want to set the environment variables for processes
	// that are spawned by the current process.
	SetEnvFunc func(key, value string)
}

// Compile time check that Carrier implements the TextMapCarrier.
var _ propagation.TextMapCarrier = (*Carrier)(nil)

// Get returns the value associated with the normalized key.
// It reads the normalized key directly from the current process environment.
func (*Carrier) Get(key string) string {
	return os.Getenv(normalize(key))
}

// Set stores the key-value pair in the environment variable.
// The key is normalized before being used to set the
// environment variable.
// If SetEnvFunc is not set, this method does nothing.
func (c *Carrier) Set(key, value string) {
	if c.SetEnvFunc == nil {
		return
	}
	k := normalize(key)
	c.SetEnvFunc(k, value)
}

// Keys lists the normalized keys stored in this carrier.
// This returns all keys from environment variables whose names are already
// normalized.
// It reads directly from the current process environment.
func (*Carrier) Keys() []string {
	environ := os.Environ()
	keys := make([]string, 0, len(environ))
	for _, kv := range environ {
		key, _, _ := strings.Cut(kv, "=")
		if !normalized(key) {
			continue
		}
		keys = append(keys, key)
	}
	return keys
}
