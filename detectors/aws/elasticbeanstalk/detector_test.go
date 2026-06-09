// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package elasticbeanstalk

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const xrayConf = "{\"deployment_id\":23,\"version_label\":\"env-version-1234\",\"environment_name\":\"BETA\"}"

type mockFileSystem struct {
	windows  bool
	exists   bool
	path     string
	contents string
}

func (mfs *mockFileSystem) Open(path string) (io.ReadCloser, error) {
	if !mfs.exists {
		return nil, errors.New("file not found")
	}
	mfs.path = path
	return io.NopCloser(strings.NewReader(mfs.contents)), nil
}

func (mfs *mockFileSystem) IsWindows() bool {
	return mfs.windows
}

func TestWindowsPath(t *testing.T) {
	mfs := &mockFileSystem{windows: true, exists: true, contents: xrayConf}
	res, err := (&ResourceDetector{fs: mfs}).Detect(t.Context())
	require.NoError(t, err)

	assert.NotNil(t, res)
	assert.Equal(t, windowsPath, mfs.path)
}

func TestFileNotExists(t *testing.T) {
	mfs := &mockFileSystem{exists: false}
	res, err := (&ResourceDetector{fs: mfs}).Detect(t.Context())

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, resource.Empty(), res)
}

func TestFileMalformed(t *testing.T) {
	mfs := &mockFileSystem{exists: true, contents: "some overwritten value"}
	res, err := (&ResourceDetector{fs: mfs}).Detect(t.Context())

	assert.Error(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, resource.Empty(), res)
}

func TestAttributesDetectedSuccessfully(t *testing.T) {
	d := &ResourceDetector{fs: &mockFileSystem{exists: true, contents: xrayConf}}

	expected := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSElasticBeanstalk,
		semconv.DeploymentID("23"),
		semconv.DeploymentEnvironmentName("BETA"),
		semconv.ServiceVersion("env-version-1234"),
	)

	res, err := d.Detect(t.Context())

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, expected, res)
}

func TestWithAttributeFilter(t *testing.T) {
	d := &ResourceDetector{
		fs:  &mockFileSystem{exists: true, contents: xrayConf},
		cfg: config{filter: attribute.NewDenyKeysFilter(semconv.CloudProviderKey)},
	}

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	// cloud.provider must be absent.
	_, ok := res.Set().Value(semconv.CloudProviderKey)
	assert.False(t, ok, "expected cloud.provider to be absent")

	// The other four attributes must be present.
	presentAttrs := []attribute.KeyValue{
		semconv.CloudPlatformAWSElasticBeanstalk,
		semconv.DeploymentID("23"),
		semconv.DeploymentEnvironmentName("BETA"),
		semconv.ServiceVersion("env-version-1234"),
	}
	for _, kv := range presentAttrs {
		val, ok := res.Set().Value(kv.Key)
		assert.True(t, ok, "expected attribute %s to be present", kv.Key)
		assert.Equal(t, kv.Value, val)
	}
}
