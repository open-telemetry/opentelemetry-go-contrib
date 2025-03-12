// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package provider // import "go.opentelemetry.io/contrib/otelconf/internal/provider"

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const ValidationPattern = `^[a-zA-Z_][a-zA-Z0-9_]*$`

var validationRegexp = regexp.MustCompile(ValidationPattern)

func ReplaceEnvVars(input []byte) ([]byte, error) {
	re := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*[-]?.*)\}`)

	replaceEnvVars := func(input []byte) ([]byte, error) {
		var err error
		out := re.ReplaceAllFunc(input, func(s []byte) []byte {
			match := re.FindSubmatch(s)
			if len(match) < 2 {
				return s
			}
			var data []byte
			data, err = replaceEnvVar(string(match[1]))
			return data
		})
		return out, err
	}
	return replaceEnvVars(input)
}

func replaceEnvVar(uri string) ([]byte, error) {
	envVarName, defaultValuePtr := parseEnvVarURI(uri)
	if !validationRegexp.MatchString(envVarName) {
		return nil, fmt.Errorf("invalid environment variable name: %s", envVarName)
	}

	val, exists := os.LookupEnv(envVarName)
	if !exists {
		if defaultValuePtr != nil {
			val = *defaultValuePtr
		}
	}
	if len(val) == 0 {
		return nil, nil
	}

	out := []byte(val)
	if err := checkRawConfType(out); err != nil {
		return nil, fmt.Errorf("invalid value type: %w", err)
	}

	return out, nil
}

func parseEnvVarURI(uri string) (string, *string) {
	const defaultSuffix = ":-"
	if strings.Contains(uri, defaultSuffix) {
		parts := strings.SplitN(uri, defaultSuffix, 2)
		return parts[0], &parts[1]
	}
	return uri, nil
}

func checkRawConfType(val []byte) error {
	var rawConf any
	err := yaml.Unmarshal(val, &rawConf)
	if err != nil {
		return err
	}
	if rawConf == nil {
		return nil
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
