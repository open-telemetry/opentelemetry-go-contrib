// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package provider // import "go.opentelemetry.io/contrib/internal/provider"

import (
	"os"
	"regexp"
	"strings"
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
	return []byte(val)
}

func parseEnvVarURI(uri string) (string, *string) {
	const defaultSuffix = ":-"
	if strings.Contains(uri, defaultSuffix) {
		parts := strings.SplitN(uri, defaultSuffix, 2)
		return parts[0], &parts[1]
	}
	return uri, nil
}
