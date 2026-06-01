// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package env // import "go.opentelemetry.io/contrib/detectors/env"

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const envVar = "OTEL_RESOURCE_ATTRIBUTES"

const deprecatedEnvVar = "OTEL_RESOURCE"

type resourceDetector struct{}

var _ resource.Detector = (*resourceDetector)(nil)

// NewResourceDetector returns a new [resource.Detector] that detects attributes
// from the OTEL_RESOURCE_ATTRIBUTES environment variable.
func NewResourceDetector() resource.Detector {
	return &resourceDetector{}
}

func (*resourceDetector) Detect(context.Context) (*resource.Resource, error) {
	labels := strings.TrimSpace(os.Getenv(envVar))
	if labels == "" {
		labels = os.Getenv(deprecatedEnvVar)
		if labels == "" {
			return nil, nil
		}
	}
	attrs, err := initializeAttributes(labels)
	if err != nil {
		return nil, err
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}

var labelRegex = regexp.MustCompile(`\s*([[:ascii:]]{1,256}?)\s*=\s*([[:ascii:]]{0,256}?)\s*(?:,|$)`)

func initializeAttributes(s string) ([]attribute.KeyValue, error) {
	var attrs []attribute.KeyValue

	matches := labelRegex.FindAllStringSubmatchIndex(s, -1)
	for len(matches) == 0 {
		return nil, fmt.Errorf("invalid resource format: %q", s)
	}

	prevIndex := 0
	for _, match := range matches {
		// if there is any text between matches, raise an error
		if prevIndex != match[0] {
			return nil, fmt.Errorf("invalid resource format, invalid text: %q", s[prevIndex:match[0]])
		}

		key := s[match[2]:match[3]]
		value := s[match[4]:match[5]]

		var err error
		if value, err = url.QueryUnescape(value); err != nil {
			return nil, fmt.Errorf("invalid resource format in attribute: %q, err: %w", s[match[0]:match[1]], err)
		}
		attrs = append(attrs, attribute.String(key, value))

		prevIndex = match[1]
	}

	// if there is any text after the last match, raise an error
	if matches[len(matches)-1][1] != len(s) {
		return nil, fmt.Errorf("invalid resource format, invalid text: %q", s[matches[len(matches)-1][1]:])
	}

	return attrs, nil
}
