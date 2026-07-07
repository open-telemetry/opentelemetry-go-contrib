// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package heroku

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func TestNewResourceDetector(t *testing.T) {
	d := NewResourceDetector()
	assert.NotNil(t, d)
}

func TestDetect_OK(t *testing.T) {
	t.Setenv(envDynoID, "foo")
	t.Setenv(envAppID, "appid")
	t.Setenv(envAppName, "appname")
	t.Setenv(envReleaseCreatedAt, "createdat")
	t.Setenv(envReleaseVersion, "v1")
	t.Setenv(envSlugCommit, "23456")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceInstanceID("foo"),
		semconv.CloudProviderHeroku,
		semconv.HerokuAppID("appid"),
		semconv.ServiceName("appname"),
		semconv.HerokuReleaseCreationTimestamp("createdat"),
		semconv.ServiceVersion("v1"),
		semconv.HerokuReleaseCommit("23456"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_Partial(t *testing.T) {
	t.Setenv(envDynoID, "foo")
	t.Setenv(envAppID, "appid")
	t.Setenv(envAppName, "appname")
	t.Setenv(envReleaseVersion, "v1")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceInstanceID("foo"),
		semconv.CloudProviderHeroku,
		semconv.HerokuAppID("appid"),
		semconv.ServiceName("appname"),
		semconv.ServiceVersion("v1"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_MissingDynoID(t *testing.T) {
	t.Setenv(envAppID, "appid")
	t.Setenv(envAppName, "appname")
	t.Setenv(envReleaseVersion, "v1")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderHeroku,
		semconv.HerokuAppID("appid"),
		semconv.ServiceName("appname"),
		semconv.ServiceVersion("v1"),
	)
	assert.Equal(t, expected, res)
}

func TestDetect_MissingAppID(t *testing.T) {
	t.Setenv(envAppName, "appname")
	t.Setenv(envReleaseVersion, "v1")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)

	expected := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("appname"),
		semconv.ServiceVersion("v1"),
	)
	assert.Equal(t, expected, res)

	_, ok := res.Set().Value(semconv.CloudProviderKey)
	assert.False(t, ok, "expected cloud.provider to be absent without app ID")
}

func TestDetect_NotOnHeroku(t *testing.T) {
	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_EmptyEnvValues(t *testing.T) {
	t.Setenv(envDynoID, "")
	t.Setenv(envAppID, "")
	t.Setenv(envAppName, "")
	t.Setenv(envReleaseCreatedAt, "")
	t.Setenv(envReleaseVersion, "")
	t.Setenv(envSlugCommit, "")

	res, err := NewResourceDetector().Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetect_WithAttributeFilter(t *testing.T) {
	t.Setenv(envDynoID, "foo")
	t.Setenv(envAppID, "appid")
	t.Setenv(envAppName, "appname")
	t.Setenv(envReleaseVersion, "v1")

	filter := attribute.NewDenyKeysFilter(semconv.ServiceVersionKey)
	res, err := NewResourceDetector(WithAttributeFilter(filter)).Detect(t.Context())
	require.NoError(t, err)

	_, ok := res.Set().Value(semconv.ServiceVersionKey)
	assert.False(t, ok, "expected service.version to be absent")

	presentAttrs := []attribute.KeyValue{
		semconv.ServiceInstanceID("foo"),
		semconv.CloudProviderHeroku,
		semconv.HerokuAppID("appid"),
		semconv.ServiceName("appname"),
	}
	for _, kv := range presentAttrs {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected attribute %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}
