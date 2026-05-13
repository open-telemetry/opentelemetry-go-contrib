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

func Test_windowsPath(t *testing.T) {
	mfs := &mockFileSystem{windows: true, exists: true, contents: xrayConf}
	res, err := (&resourceDetector{fs: mfs}).Detect(t.Context())
	require.NoError(t, err)

	assert.NotNil(t, res)
	assert.Equal(t, windowsPath, mfs.path)
}

func Test_fileNotExists(t *testing.T) {
	mfs := &mockFileSystem{exists: false}
	res, err := (&resourceDetector{fs: mfs}).Detect(t.Context())

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, resource.Empty(), res)
}

func Test_fileMalformed(t *testing.T) {
	mfs := &mockFileSystem{exists: true, contents: "some overwritten value"}
	res, err := (&resourceDetector{fs: mfs}).Detect(t.Context())

	assert.Error(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, resource.Empty(), res)
}

func Test_AttributesDetectedSuccessfully(t *testing.T) {
	d := &resourceDetector{fs: &mockFileSystem{exists: true, contents: xrayConf}}

	expected := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSElasticBeanstalk,
		semconv.ServiceInstanceID("23"),
		semconv.DeploymentEnvironmentName("BETA"),
		semconv.ServiceVersion("env-version-1234"),
	)

	res, err := d.Detect(t.Context())

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, expected, res)
}
