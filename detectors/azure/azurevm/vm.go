// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azurevm // import "go.opentelemetry.io/contrib/detectors/azure/azurevm"

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

const defaultAzureVMMetadataEndpoint = "http://169.254.169.254/metadata/instance/compute?api-version=2021-12-13&format=json"

// Azure-specific resource attribute keys that are not (yet) part of the
// semantic conventions package.
const (
	azureVMNameKey            = attribute.Key("azure.vm.name")
	azureVMSizeKey            = attribute.Key("azure.vm.size")
	azureVMScaleSetNameKey    = attribute.Key("azure.vm.scaleset.name")
	azureResourceGroupNameKey = attribute.Key("azure.resourcegroup.name")
	azureTagPrefix            = "azure.tag."
)

// ResourceDetector collects resource information of Azure VMs.
type ResourceDetector struct {
	endpoint string
	// tagKeyFilter, when non-nil, selects which VM tags are emitted as
	// azure.tag.<name> attributes by their key.
	tagKeyFilter func(key string) bool
}

type vmMetadata struct {
	VMId              *string      `json:"vmId"`
	Location          *string      `json:"location"`
	ResourceId        *string      `json:"resourceId"`
	Name              *string      `json:"name"`
	VMSize            *string      `json:"vmSize"`
	OsType            *string      `json:"osType"`
	Version           *string      `json:"version"`
	SubscriptionId    *string      `json:"subscriptionId"`
	ResourceGroupName *string      `json:"resourceGroupName"`
	VMScaleSetName    *string      `json:"vmScaleSetName"`
	Zone              *string      `json:"zone"`
	OsProfile         *osProfile   `json:"osProfile"`
	TagsList          []computeTag `json:"tagsList"`
}

type osProfile struct {
	ComputerName *string `json:"computerName"`
}

type computeTag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Option configures a [ResourceDetector].
type Option func(*ResourceDetector)

// WithTagKeyFilter emits an azure.tag.<name> attribute for every VM tag whose
// key satisfies filter. Without it, no VM tags are emitted. For regexp
// matching, pass re.MatchString.
func WithTagKeyFilter(filter func(key string) bool) Option {
	return func(d *ResourceDetector) {
		d.tagKeyFilter = filter
	}
}

// New returns a [ResourceDetector] that will detect Azure VM resources.
func New(opts ...Option) *ResourceDetector {
	d := &ResourceDetector{endpoint: defaultAzureVMMetadataEndpoint}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Detect detects associated resources when running on an Azure VM.
func (detector *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	jsonMetadata, runningInAzure, err := detector.getJSONMetadata(ctx)
	if err != nil {
		if !runningInAzure {
			return resource.Empty(), nil
		}

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
	// Prefer osProfile.computerName for host.name, falling back to the VM name
	// if it is unavailable (e.g., VMs created from specialized disks).
	if metadata.OsProfile != nil && metadata.OsProfile.ComputerName != nil && *metadata.OsProfile.ComputerName != "" {
		attributes = append(attributes, semconv.HostName(*metadata.OsProfile.ComputerName))
	} else if metadata.Name != nil {
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
	if metadata.SubscriptionId != nil {
		attributes = append(attributes, semconv.CloudAccountID(*metadata.SubscriptionId))
	}
	if metadata.Zone != nil && *metadata.Zone != "" {
		attributes = append(attributes, semconv.CloudAvailabilityZone(*metadata.Zone))
	}
	if metadata.Name != nil {
		attributes = append(attributes, azureVMNameKey.String(*metadata.Name))
	}
	if metadata.VMSize != nil {
		attributes = append(attributes, azureVMSizeKey.String(*metadata.VMSize))
	}
	if metadata.VMScaleSetName != nil && *metadata.VMScaleSetName != "" {
		attributes = append(attributes, azureVMScaleSetNameKey.String(*metadata.VMScaleSetName))
	}
	if metadata.ResourceGroupName != nil {
		attributes = append(attributes, azureResourceGroupNameKey.String(*metadata.ResourceGroupName))
	}

	if detector.tagKeyFilter != nil {
		for _, tag := range metadata.TagsList {
			if detector.tagKeyFilter(tag.Name) {
				attributes = append(attributes, attribute.String(azureTagPrefix+tag.Name, tag.Value))
			}
		}
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

func (detector *ResourceDetector) getJSONMetadata(ctx context.Context) ([]byte, bool, error) {
	pTransport := &http.Transport{Proxy: nil}

	client := http.Client{Transport: pTransport}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detector.endpoint, http.NoBody)
	if err != nil {
		return nil, false, err
	}

	req.Header.Add("Metadata", "True")

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bytes, err := io.ReadAll(resp.Body)
		return bytes, true, err
	}

	runningInAzure := resp.StatusCode < 400 || resp.StatusCode > 499

	return nil, runningInAzure, errors.New(http.StatusText(resp.StatusCode))
}
