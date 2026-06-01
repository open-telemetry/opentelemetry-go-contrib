// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	assert.NotNil(t, d)
}

func TestDetectTrue(t *testing.T) {
	t.Setenv(envVar, "key=value")
	expected := resource.NewWithAttributes(semconv.SchemaURL, attribute.String("key", "value"))

	d := NewResourceDetector()
	res, err := d.Detect(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestDetectFalse(t *testing.T) {
	d := NewResourceDetector()
	res, err := d.Detect(t.Context())
	assert.NoError(t, err)
	assert.Nil(t, res)
}

func TestDetectDeprecatedEnvVar(t *testing.T) {
	t.Setenv(deprecatedEnvVar, "key=value")
	expected := resource.NewWithAttributes(semconv.SchemaURL, attribute.String("key", "value"))

	d := NewResourceDetector()
	res, err := d.Detect(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestDetectError(t *testing.T) {
	t.Setenv(envVar, "key=value, key")

	d := NewResourceDetector()

	res, err := d.Detect(t.Context())
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestInitializeAttributes(t *testing.T) {
	cases := []struct {
		name               string
		input              string
		expectedAttributes []attribute.KeyValue
		expectedError      string
	}{
		{
			name:  "multiple valid attributes",
			input: ` example.org/test-1 =  test $ %3A \" ,  Abc=Def  `,
			expectedAttributes: []attribute.KeyValue{
				attribute.String("example.org/test-1", `test $ : \"`),
				attribute.String("Abc", "Def"),
			},
		}, {
			name:  "single valid attribute",
			input: `single=key`,
			expectedAttributes: []attribute.KeyValue{
				attribute.String("single", "key"),
			},
		}, {
			name:          "invalid url escape sequence in value",
			input:         `invalid=url-%3-encoding`,
			expectedError: `invalid resource format in attribute: "invalid=url-%3-encoding", err: invalid URL escape "%3-"`,
		}, {
			name:          "invalid char in key",
			input:         `invalid-char-ü=test`,
			expectedError: `invalid resource format: "invalid-char-ü=test"`,
		}, {
			name:          "invalid char in value",
			input:         `invalid-char=ü-test`,
			expectedError: `invalid resource format: "invalid-char=ü-test"`,
		}, {
			name:          "invalid attribute",
			input:         `extra=chars, a`,
			expectedError: `invalid resource format, invalid text: " a"`,
		}, {
			name:          "invalid char between attributes",
			input:         `invalid=char,übetween=attributes`,
			expectedError: `invalid resource format, invalid text: "ü"`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			attrs, err := initializeAttributes(c.input)
			if c.expectedError != "" {
				assert.EqualError(t, err, c.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedAttributes, attrs)
			}
		})
	}
}
