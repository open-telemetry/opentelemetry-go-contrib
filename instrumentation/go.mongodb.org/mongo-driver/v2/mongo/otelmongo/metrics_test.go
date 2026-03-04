// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelmongo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const (
	testAddr = "mongodb://localhost:27017/?connect=direct"
)

func TestMetricsOperationDuration(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	md := drivertest.NewMockDeployment()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*3)
	defer cancel()

	opts := options.Client()
	opts.Deployment = md //nolint:staticcheck // This method is the current documented way to set the mongodb mock. See https://github.com/mongodb/mongo-go-driver/blob/v2.0.0/x/mongo/driver/drivertest/opmsg_deployment_test.go#L24
	opts.Monitor = NewMonitor(
		WithMeterProvider(provider),
		WithCommandAttributeDisabled(false),
	)
	opts.ApplyURI(testAddr)

	md.AddResponses([]bson.D{{{Key: "ok", Value: 1}}}...)
	client, err := mongo.Connect(opts)
	require.NoError(t, err)
	defer func() {
		err := client.Disconnect(t.Context())
		require.NoError(t, err)
	}()

	// Perform an insert operation
	_, err = client.Database("test-database").Collection("test-collection").InsertOne(ctx, bson.D{{Key: "test-item", Value: "test-value"}})
	require.NoError(t, err)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	// Verify metrics were recorded
	require.Len(t, rm.ScopeMetrics, 1)
	scopeMetrics := rm.ScopeMetrics[0]
	assert.Equal(t, ScopeName, scopeMetrics.Scope.Name)

	// Find the operation duration metric
	var foundDuration bool
	for _, m := range scopeMetrics.Metrics {
		if m.Name != "db.client.operation.duration" {
			continue
		}
		foundDuration = true
		histogram, ok := m.Data.(metricdata.Histogram[float64])
		assert.True(t, ok, "expected histogram data type")
		assert.NotEmpty(t, histogram.DataPoints)

		// Check that attributes are present
		dp := histogram.DataPoints[0]
		attrs := dp.Attributes.ToSlice()
		hasDBSystem := false
		hasOperation := false
		for _, attr := range attrs {
			if attr.Key == "db.system.name" && attr.Value.AsString() == "mongodb" {
				hasDBSystem = true
			}
			if attr.Key == "db.operation.name" && attr.Value.AsString() == "insert" {
				hasOperation = true
			}
		}
		assert.True(t, hasDBSystem, "expected db.system.name attribute")
		assert.True(t, hasOperation, "expected db.operation.name attribute")
	}
	assert.True(t, foundDuration, "expected db.client.operation.duration metric")
}

func TestMetricsOperationFailure(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	md := drivertest.NewMockDeployment()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*3)
	defer cancel()

	opts := options.Client()
	opts.Deployment = md //nolint:staticcheck // This method is the current documented way to set the mongodb mock. See https://github.com/mongodb/mongo-go-driver/blob/v2.0.0/x/mongo/driver/drivertest/opmsg_deployment_test.go#L24
	opts.Monitor = NewMonitor(
		WithMeterProvider(provider),
		WithCommandAttributeDisabled(true),
	)
	opts.ApplyURI(testAddr)

	// Simulate an error response
	md.AddResponses([]bson.D{{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "test error"}}}...)
	client, err := mongo.Connect(opts)
	require.NoError(t, err)
	defer func() {
		err := client.Disconnect(t.Context())
		require.NoError(t, err)
	}()

	_, err = client.Database("test-database").Collection("test-collection").InsertOne(ctx, bson.D{{Key: "test-item", Value: "test-value"}})
	require.Error(t, err)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	// Verify metrics were recorded even for failed operations
	require.Len(t, rm.ScopeMetrics, 1)
	scopeMetrics := rm.ScopeMetrics[0]
	assert.NotEmpty(t, scopeMetrics.Metrics)
}

func TestNewMonitorWithInvalidMeterProvider(t *testing.T) {
	// This test verifies that NewMonitor handles errors gracefully
	// even if metric creation fails. The function should not panic
	// and should return a valid monitor that can be used.

	// Using a nil meter provider will use the global one, which should work
	monitor := NewMonitor()
	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.Started)
	assert.NotNil(t, monitor.Succeeded)
	assert.NotNil(t, monitor.Failed)
}
