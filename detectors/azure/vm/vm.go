// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vm // import "go.opentelemetry.io/contrib/detectors/azure/vm"

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const (
	defaultAzureVMMetadataEndpoint = "http://169.254.169.254/metadata/instance/compute?api-version=2021-12-13&format=json"
)

type config struct {
	endpoint string
}

func newConfig(options ...Option) config {
	c := config{defaultAzureVMMetadataEndpoint}
	for _, option := range options {
		c = option.apply(c)
	}

	return c
}

// Option applies an Azure VM detector configuration option.
type Option interface {
	apply(config) config
}

type optionFunc func(config) config

func (fn optionFunc) apply(c config) config {
	return fn(c)
}

// WithEndpoint sets the endpoint for obtaining a Azure instance metadata JSON.
func WithEndpoint(e string) Option {
	return optionFunc(func(c config) config {
		c.endpoint = e

		return c
	})
}

type resourceDetector struct {
	endpoint string
}

type vmMetadata struct {
	VMId       *string `json:"vmId"`
	Location   *string `json:"location"`
	ResourceId *string `json:"resourceId"`
	Name       *string `json:"name"`
	VMSize     *string `json:"vmSize"`
	OsType     *string `json:"osType"`
	Version    *string `json:"version"`
}

// New returns a [resource.Detector] that will detect Azure VM resources.
func New(opts ...Option) resource.Detector {
	c := newConfig(opts...)
	return &resourceDetector{c.endpoint}
}

// Detect detects associated resources when running on an Azure VM.
func (detector *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	jsonMetadata, err := detector.getJSONMetadata()
	if err != nil {
		return nil, err
	}

	var metadata vmMetadata
	err = json.Unmarshal(jsonMetadata, &metadata)
	if err != nil {
		return nil, err
	}

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAzure,
		semconv.CloudPlatformAzureVM,
	}

	if metadata.VMId != nil {
		attributes = append(attributes, semconv.HostID(*metadata.VMId))
	}
	if metadata.Location != nil {
		attributes = append(attributes, semconv.CloudRegion(*metadata.Location))
	}
	if metadata.ResourceId != nil {
		attributes = append(attributes, semconv.CloudResourceID(*metadata.ResourceId))
	}
	if metadata.Name != nil {
		attributes = append(attributes, semconv.HostName(*metadata.Name))
	}
	if metadata.VMSize != nil {
		attributes = append(attributes, semconv.HostType(*metadata.VMSize))
	}
	if metadata.OsType != nil {
		attributes = append(attributes, semconv.OSTypeKey.String(*metadata.OsType))
	}
	if metadata.Version != nil {
		attributes = append(attributes, semconv.OSVersion(*metadata.Version))
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

func (detector *resourceDetector) getJSONMetadata() ([]byte, error) {
	pTransport := &http.Transport{Proxy: nil}

	client := http.Client{Transport: pTransport}

	req, err := http.NewRequest("GET", detector.endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Metadata", "True")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
