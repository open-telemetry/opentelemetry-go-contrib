// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package provider contains various providers
// used to replace variables in configuration files.
package provider

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

const validationPattern = `^[a-zA-Z_][a-zA-Z0-9_]*$`

// stringStyles are the scalar styles that always resolve to a string.
const stringStyles = yaml.SingleQuotedStyle | yaml.DoubleQuotedStyle | yaml.LiteralStyle | yaml.FoldedStyle

var (
	validationRegexp        = regexp.MustCompile(validationPattern)
	doubleDollarSignsRegexp = regexp.MustCompile(`\$\$([^{$])`)
	envVarRegexp            = regexp.MustCompile(`([$]*)\{([a-zA-Z_][a-zA-Z0-9_]*-?[^}]*)\}`)
)

// ReplaceEnvVars parses input as YAML and substitutes environment variable
// references in its scalar values. Substitution happens on the parsed nodes,
// so a value cannot add YAML structure to the document. Mapping keys are not
// substituted.
func ReplaceEnvVars(input []byte) (*yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(input, &root); err != nil {
		return nil, err
	}
	if err := replaceNode(&root); err != nil {
		return nil, err
	}
	return &root, nil
}

func replaceNode(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		return replaceScalar(n)
	case yaml.MappingNode:
		for i := 1; i < len(n.Content); i += 2 {
			if err := replaceNode(n.Content[i]); err != nil {
				return err
			}
		}
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range n.Content {
			if err := replaceNode(c); err != nil {
				return err
			}
		}
	}
	// An alias shares the node of its anchor, which is substituted where it
	// is defined. Walking it again would apply $$ escapes twice.
	return nil
}

func replaceScalar(n *yaml.Node) error {
	if !strings.Contains(n.Value, "$") {
		return nil
	}

	val, err := replaceEnvVarsInValue([]byte(n.Value))
	if err != nil {
		return err
	}

	if n.Style&(stringStyles|yaml.TaggedStyle) != 0 {
		n.Value = string(val)
		return nil
	}

	if len(val) > 0 && isSingleReference(n.Value) {
		// The value is parsed on its own so it keeps its type, as if it had
		// been written in place of the reference.
		var doc yaml.Node
		if err := yaml.Unmarshal(val, &doc); err != nil {
			return fmt.Errorf("invalid value type: %w", err)
		}
		if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.ScalarNode {
			return errors.New("invalid value type: not a scalar")
		}
		v := doc.Content[0]
		n.Tag, n.Style, n.Value = v.Tag, v.Style, v.Value
		return nil
	}

	// Node types are interpreted after substitution, so let the decoder
	// resolve the tag from the new value.
	n.Tag = ""
	n.Value = string(val)
	return nil
}

// isSingleReference reports whether s is exactly one ${...} reference.
func isSingleReference(s string) bool {
	m := envVarRegexp.FindStringSubmatchIndex(s)
	return len(m) >= 4 && m[0] == 0 && m[1] == len(s) && m[3]-m[2] == 1
}

func replaceEnvVarsInValue(input []byte) ([]byte, error) {
	// start by replacing all $$ that are not followed by a {

	out := doubleDollarSignsRegexp.ReplaceAllFunc(input, func(s []byte) []byte {
		return append([]byte("$"), doubleDollarSignsRegexp.FindSubmatch(s)[1]...)
	})

	var err error

	out = envVarRegexp.ReplaceAllFunc(out, func(s []byte) []byte {
		// ReplaceAllFunc cannot stop early, so do not let a later match
		// overwrite the first error.
		if err != nil {
			return nil
		}

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
			data, err = replaceEnvVarsInValue(append(match[2], byte('}')))
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
	if before, after, ok := strings.Cut(in, sep); ok {
		return before, defaultValue{data: after, valid: true}
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
				" ensure that values are wrapped in quotes", rawConf,
		)
	}
}
