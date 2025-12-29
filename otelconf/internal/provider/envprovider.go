// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package provider contains various providers
// used to replace variables in configuration files.
package provider // import "go.opentelemetry.io/contrib/otelconf/internal/provider"

import (
	"bytes"
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
	// parse input file into YAML parse tree
	var tree yaml.Node

	err := yaml.Unmarshal(input, &tree)
	if err != nil {
		return nil, err
	}

	if walkErr := walkTree(&tree); walkErr != nil {
		return nil, fmt.Errorf("could not substitute environment variables: %w", walkErr)
	}

	// the result is the again serialized tree, removed a trailing newline
	result, resultErr := yaml.Marshal(&tree)
	result = bytes.TrimSuffix(result, []byte("\n"))

	if resultErr != nil {
		return nil, fmt.Errorf("could not reserialize YAML tree: %w", resultErr)
	}

	return result, nil
}

// walkTree recursively traverses the YAML parse tree and replaces environment variables in scalar nodes.
func walkTree(node *yaml.Node) error {
	if len(node.Content) == 0 &&
		node.Kind == yaml.ScalarNode {

		return handleValueNode(node)
	}

	for idx, child := range node.Content {
		if child == nil {
			continue
		}

		if node.Kind == yaml.MappingNode && idx%2 == 0 {
			// jumping over keys
			continue
		}

		err := walkTree(child)
		if err != nil {
			return err
		}
	}

	return nil
}

// handleValueNode processes a scalar node by replacing environment variables in its value.
func handleValueNode(node *yaml.Node) error {
	replace, replaceErr := ReplaceValueEnvVars([]byte(node.Value))

	if replaceErr != nil {
		return replaceErr
	}

	node.Value = string(replace)

	if node.Style != yaml.DoubleQuotedStyle &&
		node.Style != yaml.SingleQuotedStyle {
		// we had originally something like "${VALUE}", that is definitely to interpret as string
		return retypeNode(node)
	}

	return nil
}

// retypeNode analyzes the node's value to determine its YAML type and sets the appropriate tag.
func retypeNode(node *yaml.Node) error {
	// some symbols directly imply a string
	if strings.Contains(node.Value, ":") ||
		strings.Contains(node.Value, "\n") {
		node.Tag = "!!str"
		return nil
	}

	// to save the effort to parse a YAML value, we let this to the library,
	// just getting the resulting type of its parsing run
	tmpDoc := append([]byte("key: "), []byte(node.Value)...)
	var tmpNode yaml.Node

	if tmpNodeErr := yaml.Unmarshal(tmpDoc, &tmpNode); tmpNodeErr != nil {
		return fmt.Errorf("could not retype node: %w", tmpNodeErr)
	}

	if len(tmpNode.Content) != 1 || len(tmpNode.Content[0].Content) != 2 {
		return fmt.Errorf("could not retype node: %w", fmt.Errorf("unexpected node structure"))
	}

	node.Tag = tmpNode.Content[0].Content[1].Tag

	return nil
}

func ReplaceValueEnvVars(input []byte) ([]byte, error) {
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
			_, defaultValue := parseEnvVar(string(match[2]))
			if !defaultValue.valid || !strings.Contains(defaultValue.data, "$") {
				return fmt.Appendf(dollarSigns[0:(len(dollarSigns)/2)], "{%s}", match[2])
			}
			// expand the default value
			data, err = ReplaceEnvVars(append(match[2], byte('}')))
			if err != nil {
				return data
			}
			data = fmt.Appendf(dollarSigns[0:(len(dollarSigns)/2)], "{%s", data)
		}
		return data
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func replaceEnvVar(in string) ([]byte, error) {
	envVarName, defaultValue := parseEnvVar(in)
	if strings.Contains(envVarName, ":") {
		return nil, fmt.Errorf("invalid environment variable name: %s", envVarName)
	}
	if !validationRegexp.MatchString(envVarName) {
		return nil, fmt.Errorf("invalid environment variable name: %s", envVarName)
	}

	val := os.Getenv(envVarName)
	if val == "" && defaultValue.valid {
		val = strings.ReplaceAll(defaultValue.data, "$$", "$")
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

type defaultValue struct {
	data  string
	valid bool
}

func parseEnvVar(in string) (string, defaultValue) {
	in = strings.TrimPrefix(in, "env:")
	const sep = ":-"
	if i := strings.Index(in, sep); i >= 0 {
		return in[:i], defaultValue{data: in[i+len(sep):], valid: true}
	}
	return in, defaultValue{}
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
