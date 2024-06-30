// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package env // import "go.opentelemetry.io/contrib/exporters/autoexport/utils/env"

import (
	"errors"
	"os"
	"strings"
)

var (
	// ErrUndefinedVariable is returned when an environment variable is not set.
	ErrUndefinedVariable = errors.New("environment variable is undefined")
	// ErrEmptyVariable is returned when an environment variable is set but empty.
	ErrEmptyVariable = errors.New("environment variable is empty")
)

// WithStringList retrieves the value of an environment variable identified by the key
// and split it using the separator to return a list of items.
func WithStringList(key string, separator string) ([]string, error) {
	val, err := WithString(key)
	if err != nil {
		return make([]string, 0), err
	}
	return strings.Split(val, separator), nil
}

// WithDefaultString retrieves the value of an environment variable identified by the key.
// If the environment variable is not set or empty, it returns the fallback default string provided.
func WithDefaultString(key string, fallback string) string {
	val, err := WithString(key)
	if err != nil {
		return fallback
	}
	return val
}

// WithString retrieves the value of an environment variable identified by the key.
//
// ErrUndefinedVariable is returned if the environment variable lookup fails.
// ErrEmptyVariable is returned if the environment variable is empty.
func WithString(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", ErrUndefinedVariable
	}

	if val == "" {
		return "", ErrEmptyVariable
	}

	return val, nil
}
