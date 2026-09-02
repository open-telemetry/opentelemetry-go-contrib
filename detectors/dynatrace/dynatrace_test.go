// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package dynatrace

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// fullProperties holds every reported key alongside entries the detector must
// ignore: an unreported Dynatrace key, an unrelated key, and a line without a
// separator.
const fullProperties = `
dt.entity.host=my-host-from-properties
host.name=my-host-from-properties
dt.entity.host_group=my-host-group-from-properties
dt.foo=bar
dt.smartscape.host=my-smartscaped-host
invalid-entry
`

// newDetector returns a detector reading from a fresh temporary enrichment
// directory, and that directory. When content is non-empty it is written to the
// host metadata file.
func newDetector(t *testing.T, content string, opts ...Option) (*ResourceDetector, string) {
	t.Helper()

	dir := t.TempDir()
	if content != "" {
		path := filepath.Join(dir, hostMetadataFile)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	d := NewResourceDetector(opts...)
	d.enrichmentDir = dir
	return d, dir
}

func TestDetect(t *testing.T) {
	d, _ := newDetector(t, fullProperties)

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, semconv.SchemaURL, res.SchemaURL())
	assert.ElementsMatch(t, []attribute.KeyValue{
		dtEntityHostKey.String("my-host-from-properties"),
		semconv.HostName("my-host-from-properties"),
		dtSmartscapeHostKey.String("my-smartscaped-host"),
	}, res.Attributes())
}

func TestDetectSubsetOfAttributes(t *testing.T) {
	// An older OneAgent does not write dt.smartscape.host. A partial result is
	// returned without an error.
	d, _ := newDetector(t, "dt.entity.host=HOST-1\nhost.name=example.com\n")

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	assert.ElementsMatch(t, []attribute.KeyValue{
		dtEntityHostKey.String("HOST-1"),
		semconv.HostName("example.com"),
	}, res.Attributes())
}

func TestDetectValueContainsSeparator(t *testing.T) {
	// Only the first "=" separates key from value.
	d, _ := newDetector(t, "host.name=a=b=c\n")

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	assert.Equal(t, []attribute.KeyValue{semconv.HostName("a=b=c")}, res.Attributes())
}

func TestDetectTrimsWhitespace(t *testing.T) {
	d, _ := newDetector(t, "  host.name  =  example.com  \n")

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	assert.Equal(t, []attribute.KeyValue{semconv.HostName("example.com")}, res.Attributes())
}

func TestDetectSkipsEmptyKeysAndValues(t *testing.T) {
	d, _ := newDetector(t, "=orphan-value\nhost.name=\ndt.entity.host=HOST-1\n")

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	assert.Equal(t, []attribute.KeyValue{dtEntityHostKey.String("HOST-1")}, res.Attributes())
}

func TestDetectNoFile(t *testing.T) {
	// The enrichment directory exists but holds no metadata file: not running
	// on a host monitored by the OneAgent.
	d, _ := newDetector(t, "")

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectNoEnrichmentDirectory(t *testing.T) {
	d, dir := newDetector(t, "")
	d.enrichmentDir = filepath.Join(dir, "does-not-exist")

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectNoReportedKeys(t *testing.T) {
	// The file exists but holds nothing the detector reports. No schema URL is
	// emitted for an attribute-less resource.
	d, _ := newDetector(t, "dt.entity.host_group=group\ninvalid-entry\n")

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

func TestDetectUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permissions are not enforced the same way on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses file permissions")
	}

	d, dir := newDetector(t, fullProperties)
	require.NoError(t, os.Chmod(filepath.Join(dir, hostMetadataFile), 0o000))

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
	assert.ErrorIs(t, err, os.ErrPermission)
}

func TestDetectMalformedFile(t *testing.T) {
	// A line longer than the scanner buffer cannot be tokenised. The file is
	// present, so this is a read failure rather than an absent platform.
	d, _ := newDetector(t, "host.name="+strings.Repeat("a", bufio.MaxScanTokenSize)+"\n")

	res, err := d.Detect(t.Context())
	require.Error(t, err)
	assert.Nil(t, res)
	assert.ErrorIs(t, err, bufio.ErrTooLong)
}

func TestDetectWithAttributeFilter(t *testing.T) {
	keepHostName := func(kv attribute.KeyValue) bool { return kv.Key == semconv.HostNameKey }
	d, _ := newDetector(t, fullProperties, WithAttributeFilter(keepHostName))

	res, err := d.Detect(t.Context())
	require.NoError(t, err)

	assert.Equal(t, []attribute.KeyValue{
		semconv.HostName("my-host-from-properties"),
	}, res.Attributes())
}

func TestDetectWithAttributeFilterRemovingAll(t *testing.T) {
	dropAll := func(attribute.KeyValue) bool { return false }
	d, _ := newDetector(t, fullProperties, WithAttributeFilter(dropAll))

	res, err := d.Detect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, resource.Empty(), res)
}

// windowsEnrichmentDir mirrors the Windows branch of defaultEnrichmentDir for
// the given ProgramData directory.
func windowsEnrichmentDir(programData string) string {
	return filepath.Join(programData, "dynatrace", "enrichment")
}

func TestDefaultEnrichmentDir(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		programData string
		want        string
	}{
		{
			name: "linux",
			goos: "linux",
			want: "/var/lib/dynatrace/enrichment",
		},
		{
			name: "darwin",
			goos: "darwin",
			want: "/var/lib/dynatrace/enrichment",
		},
		{
			name:        "windows",
			goos:        "windows",
			programData: `D:\ProgramData`,
			want:        windowsEnrichmentDir(`D:\ProgramData`),
		},
		{
			name: "windows without ProgramData",
			goos: "windows",
			want: windowsEnrichmentDir(`C:\ProgramData`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, defaultEnrichmentDir(tt.goos, tt.programData))
		})
	}
}

func TestNewResourceDetectorUsesDefaultDir(t *testing.T) {
	d := NewResourceDetector()
	want := defaultEnrichmentDir(runtime.GOOS, os.Getenv("ProgramData"))
	assert.Equal(t, want, d.enrichmentDir)
}
