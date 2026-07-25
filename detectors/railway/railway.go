// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package railway // import "go.opentelemetry.io/contrib/detectors/railway"

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.42.0"
)

// For a complete list of Railway-provided environment variables, see:
// https://docs.railway.com/variables/reference
const (
	projectIDEnvVar       = "RAILWAY_PROJECT_ID"
	projectNameEnvVar     = "RAILWAY_PROJECT_NAME"
	environmentIDEnvVar   = "RAILWAY_ENVIRONMENT_ID"
	environmentNameEnvVar = "RAILWAY_ENVIRONMENT_NAME"
	serviceIDEnvVar       = "RAILWAY_SERVICE_ID"
	serviceNameEnvVar     = "RAILWAY_SERVICE_NAME"
	replicaIDEnvVar       = "RAILWAY_REPLICA_ID"
	replicaRegionEnvVar   = "RAILWAY_REPLICA_REGION"
	deploymentIDEnvVar    = "RAILWAY_DEPLOYMENT_ID"
	gitCommitSHAEnvVar    = "RAILWAY_GIT_COMMIT_SHA"
	gitBranchEnvVar       = "RAILWAY_GIT_BRANCH"
	gitRepoNameEnvVar     = "RAILWAY_GIT_REPO_NAME"
	gitRepoOwnerEnvVar    = "RAILWAY_GIT_REPO_OWNER"
)

// railway.* attribute keys for concepts with no equivalent generic semantic
// convention. Unlike their corresponding *_NAME variables, these IDs are
// stable across renames and can be used to link telemetry back to the
// Railway dashboard/API.
const (
	ProjectIDKey     = attribute.Key("railway.project.id")
	EnvironmentIDKey = attribute.Key("railway.environment.id")
	ServiceIDKey     = attribute.Key("railway.service.id")
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

// ResourceDetector collects resource information about the Railway service the
// process is running on.
type ResourceDetector struct {
	cfg config
}

// NewResourceDetector returns a [resource.Detector] that detects resource
// attributes on Railway.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &ResourceDetector{cfg: cfg}
}

// envAttr maps a Railway environment variable to the resource attribute
// derived from its value.
type envAttr struct {
	envVar string
	toAttr func(string) attribute.KeyValue
}

// envAttrs holds every Railway environment variable this detector maps to a
// resource attribute. See doc.go for the full list of emitted attributes and
// the variables that are intentionally left unmapped.
var envAttrs = []envAttr{
	{projectIDEnvVar, ProjectIDKey.String},
	{projectNameEnvVar, semconv.ServiceNamespace},
	{environmentIDEnvVar, EnvironmentIDKey.String},
	{environmentNameEnvVar, semconv.DeploymentEnvironmentNameKey.String},
	{serviceIDEnvVar, ServiceIDKey.String},
	{serviceNameEnvVar, semconv.ServiceName},
	{replicaIDEnvVar, semconv.ServiceInstanceID},
	{replicaRegionEnvVar, semconv.CloudRegion},
	{deploymentIDEnvVar, semconv.DeploymentID},
	{gitCommitSHAEnvVar, semconv.VCSRefHeadRevision},
	{gitBranchEnvVar, semconv.VCSRefHeadName},
	{gitRepoNameEnvVar, semconv.VCSRepositoryName},
	{gitRepoOwnerEnvVar, semconv.VCSOwnerName},
}

// Detect detects resource attributes of the Railway service the process is
// running on. It returns an empty resource and no error when not running on
// Railway.
func (d *ResourceDetector) Detect(_ context.Context) (*resource.Resource, error) {
	if os.Getenv(projectIDEnvVar) == "" {
		return resource.Empty(), nil
	}

	attrs := []attribute.KeyValue{semconv.CloudProviderKey.String("railway")}
	for _, ea := range envAttrs {
		if v := os.Getenv(ea.envVar); v != "" {
			attrs = append(attrs, ea.toAttr(v))
		}
	}

	// Railway only deploys from branches, never tags, so this is always
	// "branch" whenever a git ref is known.
	if os.Getenv(gitBranchEnvVar) != "" {
		attrs = append(attrs, semconv.VCSRefHeadTypeBranch)
	}

	// Railway's git integration is GitHub-only today.
	if os.Getenv(gitRepoOwnerEnvVar) != "" {
		attrs = append(attrs, semconv.VCSProviderNameGithub)
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

	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}
