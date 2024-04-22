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

func TestDetect(t *testing.T) {
	detector := New()

	hostResource, err := detector.Detect(context.Background())
	assert.NoError(t, err)

	attributes := []attribute.KeyValue{
		semconv.HostArchKey.String(runtime.GOARCH),
	}

	hostName, err := os.Hostname()

	// The host name is added conditionally, as it might not be available under all circumstances
	if err == nil {
		attributes = append(attributes, semconv.HostName(hostName))
	}

	// The host id is added conditionally, as it might not be available under all circumstances (for example in Windows containers)
	machineId, err := getHostId()
	if err == nil {
		attributes = append(attributes, semconv.HostID(machineId))
	}

	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)

	assert.Equal(t, expectedResource, hostResource)
}

func TestDetectWithOptIns(t *testing.T) {
	detector := New(
		WithIPAddresses(),
		WithMACAddresses(),
	)

	hostResource, err := detector.Detect(context.Background())

	assert.True(t, err == nil)

	attributes := []attribute.KeyValue{
		semconv.HostArchKey.String(runtime.GOARCH),
	}

	hostName, err := os.Hostname()

	// The host name is added conditionally, as it might not be available under all circumstances
	if err == nil {
		attributes = append(attributes, semconv.HostName(hostName))
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
