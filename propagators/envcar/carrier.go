// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package envcar // import "go.opentelemetry.io/contrib/propagators/envcar"

import (
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/propagation"
)

// Carrier is a TextMapCarrier that uses the environment variables as a
// storage medium for propagated key-value pairs. The keys are normalised
// before being used to access the environment variables.
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
	// Usually, you want to set the environment variables for processes
	// that are spawned by the current process.
	SetEnvFunc func(key, value string)
	values     map[string]string
	once       sync.Once
}

// Compile time check that Carrier implements the TextMapCarrier.
var _ propagation.TextMapCarrier = (*Carrier)(nil)

// fetch runs once on first access, and stores the environment in the
// carrier.
func (c *Carrier) fetch() {
	c.once.Do(func() {
		environ := os.Environ()
		c.values = make(map[string]string, len(environ))
		for _, kv := range environ {
			kvPair := strings.SplitN(kv, "=", 2)
			c.values[kvPair[0]] = kvPair[1]
		}
	})
}

// Get returns the value associated with the normalized passed key.
// The first call to [Carrier.Get] or [Carrier.Keys] for a
// given Carrier will read and store the values from the
// environment and all future reads will be from that store.
func (c *Carrier) Get(key string) string {
	c.fetch()
	return c.values[normalize(key)]
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

// Keys lists the keys stored in this carrier.
// This returns all the keys in the environment variables.
// The first call to [Carrier.Get] or [Carrier.Keys] for a
// given Carrier will read and store the values from the
// environment and all future reads will be from that store.
// Keys are returned as is, without any normalization, but
// this behavior is subject to change.
func (c *Carrier) Keys() []string {
	c.fetch()
	keys := make([]string, 0, len(c.values))
	for key := range c.values {
		keys = append(keys, key)
	}
	return keys
}
