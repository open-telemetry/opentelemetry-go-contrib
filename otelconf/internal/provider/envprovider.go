// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package provider contains various providers
// used to replace variables in configuration files.
package provider // import "go.opentelemetry.io/contrib/otelconf/internal/provider"

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

const validationPattern = `^[a-zA-Z_][a-zA-Z0-9_]*$`

var (
	validationRegexp        = regexp.MustCompile(validationPattern)
	doubleDollarSignsRegexp = regexp.MustCompile(`\$\$([^{$])`)
	envVarRegexp            = regexp.MustCompile(`([$]+)\{([a-zA-Z_][a-zA-Z0-9_]*-?[^}]*)\}`)
)

func ReplaceEnvVars(input []byte) ([]byte, error) {
	// parse input file into YAML parse tree
	var tree yaml.Node

	if err := yaml.Unmarshal(input, &tree); err != nil {
		return nil, err
	}

	if err := walkTree(&tree); err != nil {
		return nil, fmt.Errorf("could not substitute environment variables: %w", err)
	}

	_ = preserveYAMLLines(&tree, 0)

	// the result is the again serialized tree, removed a trailing newline
	result, err := yaml.Marshal(&tree)
	if err != nil {
		return nil, fmt.Errorf("could not reserialize YAML tree: %w", err)
	}

	result = bytes.TrimSuffix(result, []byte("\n"))

	return result, nil
}

// walkTree recursively traverses the YAML parse tree and replaces environment variables in scalar nodes.
func walkTree(node *yaml.Node) error {
	if len(node.Content) == 0 && // make sure to not run into strange situations
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
			return fmt.Errorf("error on line %d:%d: %w", child.Line, child.Column, err)
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

	// only retype if not already marked as string styled
	if node.Style != yaml.DoubleQuotedStyle &&
		node.Style != yaml.SingleQuotedStyle &&
		node.Style != yaml.FoldedStyle {
		return retypeNode(node)
	}

	// we had originally something like "${VALUE}", that is (and has been) definitely to be interpreted as string
	return nil
}

// retypeNode analyzes the node's value to determine its YAML type and sets the appropriate tag.
func retypeNode(node *yaml.Node) error {
	// some symbols directly imply a string
	if strings.Contains(node.Value, ": ") ||
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
		return fmt.Errorf("could not retype node: %w", errors.New("unexpected node structure"))
	}

	if tmpNode.Content[0].Content[1].Value != node.Value {
		// the interpretation of the value has a different length, e.g., due to tags
		node.Tag = "!!str"
	} else {
		// we got the same content, but now we have the corrected tag of the YAML parser
		node.Tag = tmpNode.Content[0].Content[1].Tag
	}

	return nil
}

func countLines(s string) int {
	if s == "" {
		return 0
	}

	return strings.Count(s, "\n") + 1
}

// preserveYAMLLines extends or inserts comments so that originally present newlines are replaced with comments.
// This slight modification preserves the line numbers of the original file as good as possible for common variable
// extension. However, when variables are extended to multiple lines, line numbers will shift upwards.
func preserveYAMLLines(node *yaml.Node, lastLine int) int {
	// Update Head comments to include possible newlines
	if node.Kind == yaml.ScalarNode {
		startLine := node.Line - countLines(node.HeadComment)

		node.HeadComment = fmt.Sprintf("%s%s",
			strings.Repeat("NL\n", max(0, startLine-lastLine-1)),
			node.HeadComment)
	}

	// Determine the new last line
	switch node.Kind {
	case yaml.DocumentNode:
		lastLine = countLines(node.HeadComment)

	case yaml.ScalarNode:
		lastLine = node.Line + max(0, countLines(node.Value)-1)
	default:
		lastLine = node.Line
	}

	// iterating over child nodes to find the last line of the (possible) compound element
	for _, child := range node.Content {
		lastLine = preserveYAMLLines(child, lastLine)
	}

	return lastLine + countLines(node.FootComment)
}

func ReplaceValueEnvVars(input []byte) ([]byte, error) {
	// start by replacing all $$ that are not followed by a $ or {
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
			data, err = ReplaceValueEnvVars(append(match[2], byte('}')))
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
