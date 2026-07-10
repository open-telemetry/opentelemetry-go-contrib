// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker // import "go.opentelemetry.io/contrib/detectors/docker"

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/moby/moby/client"
)

type hostInfo struct {
	Name   string
	OSType string
}

type containerInfo struct {
	Name      string
	ImageName *string  // for nil check if absent
	ImageID   string   // e.g. "sha256:<64-hex>"; the resolved image ID, always present
	Tags      []string // empty when the image reference has no tag, e.g. referenced by digest
}

// splitImageRef splits a Docker image reference into its name and tag,
// mirroring the distribution/reference grammar without depending on that
// module: any "@" digest is stripped before looking for a tag, and a ":"
// only introduces a tag if it appears after the last "/" (otherwise it is a
// registry host:port, e.g. "localhost:5000/name").
func splitImageRef(ref string) (name, tag string) {
	if i := strings.Index(ref, "@"); i != -1 {
		ref = ref[:i] // strip the digest before looking for a tag
	}
	rest := ref
	if slash := strings.LastIndex(ref, "/"); slash != -1 {
		rest = ref[slash+1:]
	}
	if i := strings.LastIndex(rest, ":"); i != -1 {
		return ref[:len(ref)-len(rest)+i], rest[i+1:]
	}
	return ref, ""
}

type provider interface {
	Info(context.Context) (hostInfo, error)
	ContainerInfo(context.Context) (containerInfo, error)
	Close() error
}

type dockerProviderImpl struct {
	dockerClient *client.Client
}

func (d *dockerProviderImpl) Info(ctx context.Context) (hostInfo, error) {
	result, err := d.dockerClient.Info(ctx, client.InfoOptions{})
	if err != nil {
		return hostInfo{}, err
	}
	return hostInfo{Name: result.Info.Name, OSType: result.Info.OSType}, nil
}

func (d *dockerProviderImpl) ContainerInfo(ctx context.Context) (containerInfo, error) {
	// Docker sets the container hostname to the first 12 characters of the
	// container ID by default, which ContainerInspect can resolve. This breaks
	// when a custom hostname is set via --hostname or an orchestrator (e.g.
	// Kubernetes pod.spec.hostname), in which case ContainerInspect returns
	// "no such container" (a NotFound error) and Detect treats this the same
	// as not running in a Docker container: an empty resource, no error.
	hostname, err := os.Hostname()
	if err != nil {
		return containerInfo{}, err
	}
	result, err := d.dockerClient.ContainerInspect(ctx, hostname, client.ContainerInspectOptions{})
	if err != nil {
		return containerInfo{}, fmt.Errorf("failed to fetch container information: %w", err)
	}

	var (
		imageName *string
		tags      []string
	)
	// Config.Image is the reference the operator supplied when creating the
	// container; Container.Image is always the resolved image digest. Bare image
	// ID references may be the full digest, the hex ID without the algorithm
	// prefix, or a unique prefix, so suppress name/tag for those forms.
	if result.Container.Config != nil && result.Container.Config.Image != "" {
		ref := result.Container.Config.Image
		id := strings.TrimPrefix(result.Container.Image, "sha256:")
		refID := strings.TrimPrefix(strings.ToLower(ref), "sha256:")
		isBareImageID := refID != "" && len(refID) >= 12 && len(refID) <= len(id) && strings.HasPrefix(id, refID)
		for _, c := range refID {
			isHexDigit := ('0' <= c && c <= '9') || ('a' <= c && c <= 'f')
			if !isHexDigit {
				isBareImageID = false
				break
			}
		}
		if !isBareImageID {
			name, tag := splitImageRef(ref)
			imageName = &name
			if tag != "" {
				tags = []string{tag}
			}
		}
	}
	return containerInfo{
		Name:      strings.TrimPrefix(result.Container.Name, "/"),
		ImageName: imageName,
		ImageID:   result.Container.Image,
		Tags:      tags,
	}, nil
}

// Close releases the resources held by the underlying Docker client, such as
// idle connections to the daemon.
func (d *dockerProviderImpl) Close() error {
	return d.dockerClient.Close()
}

func newProvider(opts ...client.Opt) (provider, error) {
	opts = append([]client.Opt{client.FromEnv}, opts...)
	cli, err := client.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("could not initialize Docker client: %w", err)
	}
	return &dockerProviderImpl{dockerClient: cli}, nil
}
