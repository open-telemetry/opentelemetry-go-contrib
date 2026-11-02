// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package dynatrace

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	// hostMetadataFile is the enrichment file the OneAgent writes.
	hostMetadataFile = "dt_host_metadata.properties"

	// unixEnrichmentDir is the enrichment directory on non-Windows systems.
	unixEnrichmentDir = "/var/lib/dynatrace/enrichment"

	// fallbackProgramData is used on Windows when the ProgramData environment
	// variable is not set.
	fallbackProgramData = `C:\ProgramData`
)

// Dynatrace-specific attribute keys. These identify the host entity in
// Dynatrace and have no equivalent in the OpenTelemetry semantic conventions,
// so they are declared here rather than taken from semconv.
const (
	dtEntityHostKey     = attribute.Key("dt.entity.host")
	dtSmartscapeHostKey = attribute.Key("dt.smartscape.host")
)

// Compile-time interface assertion.
var _ resource.Detector = (*ResourceDetector)(nil)

type config struct {
	filter attribute.Filter
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithAttributeFilter sets a filter that controls which detected attributes are
// included in the returned resource. Only attributes for which filter returns
// true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// ResourceDetector collects resource information from the enrichment files
// written by the Dynatrace OneAgent.
type ResourceDetector struct {
	enrichmentDir string
	cfg           config
}

// NewResourceDetector returns a [resource.Detector] that reads the Dynatrace
// OneAgent host enrichment file.
//
// If the file is absent, i.e. the process is not running on a host monitored by
// the OneAgent, the detector returns an empty resource without an error.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{
		enrichmentDir: defaultEnrichmentDir(runtime.GOOS, os.Getenv("ProgramData")),
		cfg:           cfg,
	}
}

// defaultEnrichmentDir returns the directory the OneAgent writes its enrichment
// files to on goos. On Windows the directory lives under programData, falling
// back to C:\ProgramData when the ProgramData environment variable is unset.
func defaultEnrichmentDir(goos, programData string) string {
	if goos != "windows" {
		return unixEnrichmentDir
	}
	if programData == "" {
		programData = fallbackProgramData
	}
	return filepath.Join(programData, "dynatrace", "enrichment")
}

// Detect reads the Dynatrace OneAgent host enrichment file and returns the
// attributes it declares. It returns an empty resource and no error when the
// file does not exist, and an error when the file exists but cannot be read.
//
// All attributes are optional: a resource holding only the subset of attributes
// present in the file is returned without an error, so
// [resource.ErrPartialResource] is never returned.
func (d *ResourceDetector) Detect(context.Context) (*resource.Resource, error) {
	path := filepath.Join(d.enrichmentDir, hostMetadataFile)

	file, err := os.Open(path) // #nosec G304 -- path is derived from a fixed, non-user-supplied directory.
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Not running on a host monitored by the OneAgent.
			return resource.Empty(), nil
		}
		return nil, fmt.Errorf("dynatrace: open %s: %w", path, err)
	}
	defer file.Close()

	attrs, err := parseProperties(file)
	if err != nil {
		return nil, fmt.Errorf("dynatrace: read %s: %w", path, err)
	}

	if d.cfg.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if d.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	if len(attrs) == 0 {
		// Returning a resource carrying only a schema URL would risk a schema
		// conflict when merged with other detectors for no gain.
		return resource.Empty(), nil
	}
	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}

// parseProperties reads key=value lines from r and returns the attributes for
// the keys the detector reports. Lines that do not hold a "=" separator, and
// keys that are not reported by the detector, are skipped.
//
// This deliberately mirrors the collector implementation rather than
// implementing a full Java properties parser: comments, escape sequences and
// the ":" separator are not recognized.
func parseProperties(r io.Reader) ([]attribute.KeyValue, error) {
	var attrs []attribute.KeyValue

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		// Split on the first "=" only; any later "=" is part of the value.
		key, value, found := strings.Cut(scanner.Text(), "=")
		if !found {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}

		switch key {
		case string(dtEntityHostKey):
			attrs = append(attrs, dtEntityHostKey.String(value))
		case string(semconv.HostNameKey):
			attrs = append(attrs, semconv.HostName(value))
		case string(dtSmartscapeHostKey):
			attrs = append(attrs, dtSmartscapeHostKey.String(value))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return attrs, nil
}
