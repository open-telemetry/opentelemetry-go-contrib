// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package provider // import "go.opentelemetry.io/contrib/internal/provider"

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

func ReplaceEnvVar(uri string) []byte {
	envVarName, defaultValuePtr := parseEnvVarURI(uri)
	if !validationRegexp.MatchString(envVarName) {
		return nil
	}

	val, exists := os.LookupEnv(envVarName)
	if !exists {
		if defaultValuePtr != nil {
			val = *defaultValuePtr
		}
	}
	if len(val) == 0 {
		return nil
	}

	out := []byte(val)
	if err := checkRawConfType(out); err != nil {
		return nil
	}

	return out
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
