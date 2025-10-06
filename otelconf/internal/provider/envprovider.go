// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package provider contains various providers
// used to replace variables in configuration files.
package provider // import "go.opentelemetry.io/contrib/otelconf/internal/provider"

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

const validationPattern = `^[a-zA-Z_][a-zA-Z0-9_]*$`

var (
	validationRegexp        = regexp.MustCompile(validationPattern)
	doubleDollarSignsRegexp = regexp.MustCompile(`\$\$([^{$])`)
	envVarRegexp            = regexp.MustCompile(`([$]*)\{([a-zA-Z_][a-zA-Z0-9_]*-?[^}]*)\}`)
)

func ReplaceEnvVars(input []byte) ([]byte, error) {
	// start by replacing all $$ that are not followed by a {

	out := doubleDollarSignsRegexp.ReplaceAllFunc(input, func(s []byte) []byte {
		return append([]byte("$"), doubleDollarSignsRegexp.FindSubmatch(s)[1]...)
	})

	var err error

	out = envVarRegexp.ReplaceAllFunc(out, func(s []byte) []byte {
		match := envVarRegexp.FindSubmatch(s)
		var data []byte

		// check if we have an odd number of $, which indicates that
		// env var replacement should be done
		dollarSigns := match[1]
		if len(match) > 2 && (len(dollarSigns)%2 == 1) {
			data, err = replaceEnvVar(string(match[2]))
			if err != nil {
				return data
			}
			if len(dollarSigns) > 1 {
				data = append(dollarSigns[0:(len(dollarSigns)/2)], data...)
			}
		} else {
			// need to expand any default value env var to support the case $${STRING_VALUE:-${STRING_VALUE}}
			_, defaultValuePtr := parseEnvVar(string(match[2]))
			if defaultValuePtr == nil || !strings.Contains(*defaultValuePtr, "$") {
				return append(dollarSigns[0:(len(dollarSigns)/2)], []byte(fmt.Sprintf("{%s}", match[2]))...)
			}
			// expand the default value
			data, err = ReplaceEnvVars(append(match[2], byte('}')))
			if err != nil {
				return data
			}
			data = append(dollarSigns[0:(len(dollarSigns)/2)], []byte(fmt.Sprintf("{%s", data))...)
		}
		return data
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func replaceEnvVar(in string) ([]byte, error) {
	envVarName, defaultValuePtr := parseEnvVar(in)
	if strings.Contains(envVarName, ":") {
		return nil, fmt.Errorf("invalid environment variable name: %s", envVarName)
	}
	if !validationRegexp.MatchString(envVarName) {
		return nil, fmt.Errorf("invalid environment variable name: %s", envVarName)
	}

	val := os.Getenv(envVarName)
	if val == "" && defaultValuePtr != nil {
		val = strings.ReplaceAll(*defaultValuePtr, "$$", "$")
	}
	if val == "" {
		return nil, nil
	}

	out := []byte(val)
	if err := checkRawConfType(out); err != nil {
		return nil, fmt.Errorf("invalid value type: %w", err)
	}

	return out, nil
}

func parseEnvVar(in string) (string, *string) {
	in = strings.TrimPrefix(in, "env:")
	const defaultSuffix = ":-"
	if strings.Contains(in, defaultSuffix) {
		parts := strings.SplitN(in, defaultSuffix, 2)
		return parts[0], &parts[1]
	}
	return in, nil
}

func checkRawConfType(val []byte) error {
	var rawConf any
	err := yaml.Unmarshal(val, &rawConf)
	if err != nil {
		return err
	}

	switch rawConf.(type) {
	case int, int32, int64, float32, float64, bool, string, time.Time:
		return nil
	default:
		return fmt.Errorf(
			"unsupported type=%T for retrieved config,"+
				" ensure that values are wrapped in quotes", rawConf)
	}
}
