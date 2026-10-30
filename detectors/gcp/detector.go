// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"cloud.google.com/go/compute/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"go.opentelemetry.io/contrib/detectors/gcp/internal"
)

// NewDetector returns a resource detector which detects resource attributes on:
//
//   - Google Compute Engine (GCE).
//   - Google Kubernetes Engine (GKE).
//   - Google App Engine (GAE).
//   - Cloud Run.
//   - Cloud Run jobs.
//   - Cloud Run worker pools.
//   - Cloud Functions.
//   - Bare Metal Solution (BMS).
func NewDetector() resource.Detector {
	return NewDetectorWithOptions()
}

// NewDetectorWithOptions returns a resource detector configured with the provided options.
func NewDetectorWithOptions(opts ...Option) resource.Detector {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return &detector{detector: internal.NewDetector(), cfg: cfg}
}

// Option configures a GCP resource detector.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

type config struct {
	gkeHostType       bool
	gceLabelKeyFilter func(key string) bool
}

// WithGKEHostType enables detection of the host.type resource attribute on
// Google Kubernetes Engine (GKE) by querying the Compute Engine instances.get
// API. It is disabled by default because it makes an additional API call and
// requires the compute.instances.get IAM permission (as well as access to
// instance/name metadata, which is unavailable under GKE Workload Identity).
func WithGKEHostType() Option {
	return optionFunc(func(c *config) {
		c.gkeHostType = true
	})
}

// WithGCEInstanceLabelKeyFilter emits a gcp.gce.instance.labels.<key>
// attribute on Google Compute Engine (GCE) for every instance label whose key
// satisfies filter. The filter receives the raw GCE instance label key, without
// the "gcp.gce.instance.labels." prefix.
//
// It is disabled by default (when filter is nil) because it makes an
// additional call to the Compute Engine instances.get API and requires the
// compute.instances.get IAM permission. For regexp matching, pass
// re.MatchString.
func WithGCEInstanceLabelKeyFilter(filter func(key string) bool) Option {
	return optionFunc(func(c *config) {
		c.gceLabelKeyFilter = filter
	})
}

type detector struct {
	detector gcpDetector
	cfg      config
}

// Detect detects associated resources when running on GCE, GKE, GAE,
// Cloud Run, Cloud Run jobs, Cloud Run worker pools, Cloud Functions, and Bare Metal Solution.
func (d *detector) Detect(ctx context.Context) (*resource.Resource, error) {
	projectID, err1 := d.detector.BareMetalSolutionProjectID(ctx)
	region, err2 := d.detector.BareMetalSolutionCloudRegion(ctx)
	instanceID, err3 := d.detector.BareMetalSolutionInstanceID(ctx)
	if err1 == nil && err2 == nil && err3 == nil && projectID != "" && region != "" && instanceID != "" {
		return resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.CloudProviderGCP,
			semconv.CloudPlatformGCPBareMetalSolution,
			semconv.CloudAccountID(projectID),
			semconv.HostID(instanceID),
			semconv.CloudRegion(region),
		), nil
	}

	if !metadata.OnGCEWithContext(ctx) {
		return nil, nil
	}
	b := &resourceBuilder{}
	b.attrs = append(b.attrs, semconv.CloudProviderGCP)
	b.add(ctx, semconv.CloudAccountIDKey, d.detector.ProjectID)

	switch d.detector.CloudPlatform(ctx) {
	case internal.GKE:
		b.attrs = append(b.attrs, semconv.CloudPlatformGCPKubernetesEngine)
		b.addZoneOrRegion(ctx, d.detector.GKEAvailabilityZoneOrRegion)
		b.add(ctx, semconv.K8SClusterNameKey, d.detector.GKEClusterName)
		b.add(ctx, semconv.HostIDKey, d.detector.GKEHostID)
		if d.cfg.gkeHostType {
			b.add(ctx, semconv.HostTypeKey, func(ctx context.Context) (string, error) {
				return d.detector.GKEHostType(ctx)
			})
		}
	case internal.CloudRun, internal.CloudRunWorkerPool:
		b.attrs = append(b.attrs, semconv.CloudPlatformGCPCloudRun)
		b.add(ctx, semconv.FaaSNameKey, d.detector.FaaSName)
		b.add(ctx, semconv.FaaSVersionKey, d.detector.FaaSVersion)
		b.add(ctx, semconv.FaaSInstanceKey, d.detector.FaaSID)
		b.add(ctx, semconv.CloudRegionKey, d.detector.FaaSCloudRegion)
	case internal.CloudRunJob:
		b.attrs = append(b.attrs, semconv.CloudPlatformGCPCloudRun)
		b.add(ctx, semconv.FaaSNameKey, d.detector.FaaSName)
		b.add(ctx, semconv.FaaSInstanceKey, d.detector.FaaSID)
		b.add(ctx, semconv.GCPCloudRunJobExecutionKey, d.detector.CloudRunJobExecution)
		b.addInt(ctx, semconv.GCPCloudRunJobTaskIndexKey, d.detector.CloudRunJobTaskIndex)
		b.add(ctx, semconv.CloudRegionKey, d.detector.FaaSCloudRegion)
	case internal.CloudFunctions:
		b.attrs = append(b.attrs, semconv.CloudPlatformGCPCloudFunctions)
		b.add(ctx, semconv.FaaSNameKey, d.detector.FaaSName)
		b.add(ctx, semconv.FaaSVersionKey, d.detector.FaaSVersion)
		b.add(ctx, semconv.FaaSInstanceKey, d.detector.FaaSID)
		b.add(ctx, semconv.CloudRegionKey, d.detector.FaaSCloudRegion)
	case internal.AppEngineFlex:
		b.attrs = append(b.attrs, semconv.CloudPlatformGCPAppEngine)
		b.addZoneAndRegion(ctx, d.detector.AppEngineFlexAvailabilityZoneAndRegion)
		b.add(ctx, semconv.FaaSNameKey, d.detector.AppEngineServiceName)
		b.add(ctx, semconv.FaaSVersionKey, d.detector.AppEngineServiceVersion)
		b.add(ctx, semconv.FaaSInstanceKey, d.detector.AppEngineServiceInstance)
	case internal.AppEngineStandard:
		b.attrs = append(b.attrs, semconv.CloudPlatformGCPAppEngine)
		b.add(ctx, semconv.CloudAvailabilityZoneKey, d.detector.AppEngineStandardAvailabilityZone)
		b.add(ctx, semconv.CloudRegionKey, d.detector.AppEngineStandardCloudRegion)
		b.add(ctx, semconv.FaaSNameKey, d.detector.AppEngineServiceName)
		b.add(ctx, semconv.FaaSVersionKey, d.detector.AppEngineServiceVersion)
		b.add(ctx, semconv.FaaSInstanceKey, d.detector.AppEngineServiceInstance)
	case internal.GCE:
		b.attrs = append(b.attrs, semconv.CloudPlatformGCPComputeEngine)
		b.addZoneAndRegion(ctx, d.detector.GCEAvailabilityZoneAndRegion)
		b.add(ctx, semconv.HostTypeKey, d.detector.GCEHostType)
		b.add(ctx, semconv.HostIDKey, d.detector.GCEHostID)
		b.add(ctx, semconv.HostNameKey, d.detector.GCEHostName)
		b.add(ctx, semconv.GCPGCEInstanceNameKey, d.detector.GCEInstanceName)
		b.add(ctx, semconv.GCPGCEInstanceHostnameKey, d.detector.GCEInstanceHostname)
		b.addManagedInstanceGroup(ctx, d.detector.GCEManagedInstanceGroup)
		if d.cfg.gceLabelKeyFilter != nil {
			b.addGCEInstanceLabels(ctx, func(ctx context.Context) (map[string]string, error) {
				return d.detector.GCEInstanceLabels(ctx)
			}, d.cfg.gceLabelKeyFilter)
		}
	default:
		// We don't support this platform yet, so just return with what we have
	}
	return b.build()
}

// resourceBuilder simplifies constructing resources using GCP detection
// library functions.
type resourceBuilder struct {
	errs  []error
	attrs []attribute.KeyValue
}

func (r *resourceBuilder) add(ctx context.Context, key attribute.Key, detect func(context.Context) (string, error)) {
	if v, err := detect(ctx); err == nil {
		r.attrs = append(r.attrs, key.String(v))
	} else {
		r.errs = append(r.errs, err)
	}
}

func (r *resourceBuilder) addInt(ctx context.Context, key attribute.Key, detect func(context.Context) (string, error)) {
	if v, err := detect(ctx); err == nil {
		if vi, err := strconv.Atoi(v); err == nil {
			r.attrs = append(r.attrs, key.Int(vi))
		} else {
			r.errs = append(r.errs, err)
		}
	} else {
		r.errs = append(r.errs, err)
	}
}

// addZoneAndRegion functions are expected to return zone, region, err.
func (r *resourceBuilder) addZoneAndRegion(ctx context.Context, detect func(context.Context) (string, string, error)) {
	if zone, region, err := detect(ctx); err == nil {
		r.attrs = append(
			r.attrs,
			semconv.CloudAvailabilityZone(zone),
			semconv.CloudRegion(region),
		)
	} else {
		r.errs = append(r.errs, err)
	}
}

func (r *resourceBuilder) addZoneOrRegion(ctx context.Context, detect func(context.Context) (string, internal.LocationType, error)) {
	if v, locType, err := detect(ctx); err == nil {
		switch locType {
		case internal.Zone:
			r.attrs = append(r.attrs, semconv.CloudAvailabilityZone(v))
		case internal.Region:
			r.attrs = append(r.attrs, semconv.CloudRegion(v))
		default:
			r.errs = append(r.errs, fmt.Errorf("location must be zone or region. Got %v", locType))
		}
	} else {
		r.errs = append(r.errs, err)
	}
}

func (r *resourceBuilder) addManagedInstanceGroup(ctx context.Context, detect func(context.Context) (internal.ManagedInstanceGroup, error)) {
	if mig, err := detect(ctx); err == nil {
		if mig.Name != "" {
			r.attrs = append(r.attrs, semconv.GCPGCEInstanceGroupManagerName(mig.Name))
			switch mig.Type {
			case internal.Zone:
				r.attrs = append(r.attrs, semconv.GCPGCEInstanceGroupManagerZone(mig.Location))
			case internal.Region:
				r.attrs = append(r.attrs, semconv.GCPGCEInstanceGroupManagerRegion(mig.Location))
			default:
				r.errs = append(r.errs, fmt.Errorf("managed instance group location must be zone or region. Got %v", mig.Type))
			}
		}
	} else {
		r.errs = append(r.errs, err)
	}
}

func (r *resourceBuilder) addGCEInstanceLabels(ctx context.Context, detect func(context.Context) (map[string]string, error), filter func(key string) bool) {
	if labels, err := detect(ctx); err == nil {
		for k, v := range labels {
			if filter(k) {
				r.attrs = append(r.attrs, semconv.GCPGCEInstanceLabels(k, v))
			}
		}
	} else {
		r.errs = append(r.errs, err)
	}
}

func (r *resourceBuilder) build() (*resource.Resource, error) {
	var err error
	if len(r.errs) > 0 {
		err = fmt.Errorf("%w: %w", resource.ErrPartialResource, errors.Join(r.errs...))
	}
	return resource.NewWithAttributes(semconv.SchemaURL, r.attrs...), err
}
