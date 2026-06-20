// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envcar // import "go.opentelemetry.io/contrib/propagators/envcar"

import (
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/propagation"
)

// Carrier is a TextMapCarrier that uses environment variables as a storage
// medium for propagated key-value pairs. Keys passed to [Carrier.Get] and
// [Carrier.Set] are normalized before lookup or write. Environment variables
// listed by [Carrier.Keys] are stored only when their names are already
// normalized. [Carrier.Get] reads the normalized key directly.
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
	values     map[string]struct{}
	once       sync.Once
}

// Compile time check that Carrier implements the TextMapCarrier.
var _ propagation.TextMapCarrier = (*Carrier)(nil)

// fetch runs once on first Keys access, and stores environment variables with
// already-normalized names in the carrier.
func (c *Carrier) fetch() {
	c.once.Do(func() {
		environ := os.Environ()
		c.values = make(map[string]struct{}, len(environ))
		for _, kv := range environ {
			kvPair := strings.SplitN(kv, "=", 2)
			key := kvPair[0]
			if !normalized(key) {
				continue
			}
			c.values[key] = struct{}{}
		}
	})
}

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
// The first call to [Carrier.Keys] for a given Carrier will read and store the
// values from environment variables whose names are already normalized, and all
// future reads will be from that store.
func (c *Carrier) Keys() []string {
	c.fetch()
	keys := make([]string, 0, len(c.values))
	for key := range c.values {
		keys = append(keys, key)
	}
	return keys
}
