// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package elasticbeanstalk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
)

const (
	linuxPath   = "/var/elasticbeanstalk/xray/environment.conf"
	windowsPath = "C:\\Program Files\\Amazon\\XRay\\environment.conf"
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

// ResourceDetector reads the AWS X-Ray daemon configuration file to detect
// Elastic Beanstalk resource attributes.
type ResourceDetector struct {
	fs  fileSystem
	cfg config
}

// NewResourceDetector returns a resource detector that reads the AWS X-Ray daemon
// configuration file to detect Elastic Beanstalk resource attributes.
//
// If the configuration file is absent (i.e. the process is not running on an
// Elastic Beanstalk environment, or X-Ray integration is disabled), the detector
// returns an empty resource without an error.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{fs: &ebFileSystem{}, cfg: cfg}
}

type ebMetaData struct {
	DeploymentID    int    `json:"deployment_id"`
	EnvironmentName string `json:"environment_name"`
	VersionLabel    string `json:"version_label"`
}

// Detect collects resource attributes available when running on elasticbeanstalk.
func (detector *ResourceDetector) Detect(context.Context) (*resource.Resource, error) {
	var conf io.ReadCloser
	var err error
	if detector.fs.IsWindows() {
		conf, err = detector.fs.Open(windowsPath)
	} else {
		conf, err = detector.fs.Open(linuxPath)
	}

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return resource.Empty(), nil
		}
		return resource.Empty(), fmt.Errorf("elasticbeanstalk: %w", err)
	}

	defer conf.Close()

	ebmd := &ebMetaData{}
	err = json.NewDecoder(conf).Decode(ebmd)
	if err != nil {
		return resource.Empty(), err
	}

	attrs := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSElasticBeanstalk,
		semconv.DeploymentID(strconv.Itoa(ebmd.DeploymentID)),
		semconv.DeploymentEnvironmentNameKey.String(ebmd.EnvironmentName),
		semconv.ServiceVersion(ebmd.VersionLabel),
	}

	if detector.cfg.filter != nil {
		filtered := attrs[:0]
		for _, kv := range attrs {
			if detector.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attrs = filtered
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}
