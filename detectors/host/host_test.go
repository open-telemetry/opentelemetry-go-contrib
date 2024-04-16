// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package host

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func Test_Detect(t *testing.T) {
	detector := NewResourceDetector()

	hostResource, err := detector.Detect(context.Background())

	assert.True(t, err == nil)

	hostName, _ := os.Hostname()

	attributes := []attribute.KeyValue{
		semconv.HostArchKey.String(runtime.GOARCH),
		semconv.HostName(hostName),
	}

	// The host id is added conditionally, as it might not be available under all circumstances (for example in Windows containers)
	machineId, err := getHostId()
	if err == nil {
		attributes = append(attributes, semconv.HostID(machineId))
	}

	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)

	assert.Equal(t, expectedResource, hostResource)
}

func Test_Detect_WithOptIns(t *testing.T) {
	detector := NewResourceDetector(
		WithIPAddresses(),
		WithMACAddresses(),
	)

	hostResource, err := detector.Detect(context.Background())

	assert.True(t, err == nil)

	hostName, _ := os.Hostname()

	attributes := []attribute.KeyValue{
		semconv.HostArchKey.String(runtime.GOARCH),
		semconv.HostName(hostName),
	}

	// The host id is added conditionally, as it might not be available under all circumstances (for example in Windows containers)
	machineId, err := getHostId()
	if err == nil {
		attributes = append(attributes, semconv.HostID(machineId))
	}

	attributes = append(attributes, semconv.HostIP(getIPAddresses()...))
	attributes = append(attributes, semconv.HostMac(getMACAddresses()...))

	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)

	assert.Equal(t, expectedResource, hostResource)
}
