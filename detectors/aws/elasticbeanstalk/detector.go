// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package elasticbeanstalk // import "go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk"

import (
	"context"
	"encoding/json"
	"io"
	"strconv"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const (
	linuxPath   = "/var/elasticbeanstalk/xray/environment.conf"
	windowsPath = "C:\\Program Files\\Amazon\\XRay\\environment.conf"
)

type resourceDetector struct {
	fs fileSystem
}

// compile time assertion that resource detector implements the resource.Detector interface.
var _ resource.Detector = (*resourceDetector)(nil)

// NewResourceDetector returns a resource detector that will detect AWS Elastic Beanstalk resources.
func NewResourceDetector() resource.Detector {
	return &resourceDetector{fs: &ebFileSystem{}}
}

type ebMetaData struct {
	DeploymentID    int    `json:"deployment_id"`
	EnvironmentName string `json:"environment_name"`
	VersionLabel    string `json:"version_label"`
}

func (detector *resourceDetector) Detect(context.Context) (*resource.Resource, error) {
	var conf io.ReadCloser
	var err error
	if detector.fs.IsWindows() {
		conf, err = detector.fs.Open(windowsPath)
	} else {
		conf, err = detector.fs.Open(linuxPath)
	}

	// Do not want to return error so it fails silently on non-EB instances
	if err != nil {
		return resource.Empty(), nil
	}

	ebmd := &ebMetaData{}
	err = json.NewDecoder(conf).Decode(ebmd)
	conf.Close()

	if err != nil {
		// TODO: Log a more specific error with zap
		return resource.Empty(), err
	}

	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSElasticBeanstalk,
		semconv.DeploymentID(strconv.Itoa(ebmd.DeploymentID)),
		semconv.DeploymentEnvironmentName(ebmd.EnvironmentName),
		semconv.ServiceVersion(ebmd.VersionLabel),
	), nil
}
