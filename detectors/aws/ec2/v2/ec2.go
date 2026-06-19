// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package ec2 provides a resource detector for EC2 instances using aws-sdk-go-v2.
package ec2 // import "go.opentelemetry.io/contrib/detectors/aws/ec2/v2"

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/smithy-go/logging"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

var errClient = errors.New("EC2 Client Error")

// client implements methods to capture EC2 environment metadata information.
type client interface {
	GetInstanceIdentityDocument(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error)
	GetMetadata(ctx context.Context, params *imds.GetMetadataInput, optFns ...func(*imds.Options)) (*imds.GetMetadataOutput, error)
}

// resource detector collects resource information from EC2 environment.
type resourceDetector struct {
	c client
}

// compile time assertion that imds.Client implements client.
var _ client = (*imds.Client)(nil)

// compile time assertion that resourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*resourceDetector)(nil)

// NewResourceDetector returns a resource detector that will detect AWS EC2 resources.
func NewResourceDetector() resource.Detector {
	// Drop error to preserve stable API.
	client, _ := newClient(config{})
	return &resourceDetector{c: client}
}

// NewResourceDetectorWithOptions returns a resource detector that will detect
// AWS EC2 resources. The options configure the AWS SDK client.
func NewResourceDetectorWithOptions(opts ...Option) (resource.Detector, error) {
	var c config
	for _, opt := range opts {
		opt.apply(&c)
	}
	client, err := newClient(c)
	if err != nil {
		return nil, err
	}
	return &resourceDetector{c: client}, nil
}

// WithLogger passes the [logging.Logger] to the AWS SDK client.
func WithLogger(logger logging.Logger) Option {
	return optionFunc(func(c *config) {
		c.logger = logger
	})
}

// Option represents a AWS SDK client configuration option function.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

type config struct {
	logger logging.Logger
}

func (detector *resourceDetector) getClient() client {
	return detector.c
}

// Detect detects associated resources when running in AWS environment.
func (detector *resourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	// Return nil if not able to establish valid client
	client := detector.getClient()
	if client == nil {
		return nil, errClient
	}

	// Available method removed in aws-sdk-go-v2, return empty resource if client returns error
	doc, err := client.GetInstanceIdentityDocument(ctx, nil)
	if err != nil {
		return resource.Empty(), nil
	}

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEC2,
		semconv.CloudRegion(doc.Region),
		semconv.CloudAvailabilityZone(doc.AvailabilityZone),
		semconv.CloudAccountID(doc.AccountID),
		semconv.HostID(doc.InstanceID),
		semconv.HostImageID(doc.ImageID),
		semconv.HostType(doc.InstanceType),
	}

	m := &metadata{client: client}
	m.add(ctx, semconv.HostNameKey, "hostname")

	attributes = append(attributes, m.attributes...)

	if len(m.errs) > 0 {
		err = fmt.Errorf("%w: %s", resource.ErrPartialResource, m.errs)
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), err
}

// loadConfig loads the default AWS configuration. It is a package-level
// variable so tests can substitute a fake.
var loadConfig = awsconfig.LoadDefaultConfig

func newClient(c config) (client, error) {
	var optFns []func(*awsconfig.LoadOptions) error
	if c.logger != nil {
		optFns = append(optFns, awsconfig.WithLogger(c.logger))
	}

	cfg, err := loadConfig(context.Background(), optFns...)
	if err != nil {
		return nil, err
	}

	return imds.NewFromConfig(cfg), nil
}

type metadata struct {
	client     client
	errs       []error
	attributes []attribute.KeyValue
}

func (m *metadata) add(ctx context.Context, k attribute.Key, n string) {
	metadataInput := &imds.GetMetadataInput{Path: n}
	md, err := m.client.GetMetadata(ctx, metadataInput)
	if err != nil {
		m.recordError(n, err)
		return
	}
	data, err := io.ReadAll(md.Content)
	if err != nil {
		m.recordError(n, err)
		return
	}
	m.attributes = append(m.attributes, k.String(string(data)))
}

func (m *metadata) recordError(path string, err error) {
	var rf *awshttp.ResponseError
	ok := errors.As(err, &rf)
	if !ok {
		m.errs = append(m.errs, fmt.Errorf("%q: %w", path, err))
		return
	}

	if rf.HTTPStatusCode() == http.StatusNotFound {
		return
	}

	m.errs = append(m.errs, fmt.Errorf("%q: %d %s", path, rf.HTTPStatusCode(), rf.Error()))
}
