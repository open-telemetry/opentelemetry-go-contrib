// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package eks provides a resource detector for AWS EKS.
package eks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"k8s.io/client-go/rest"
)

const (
	k8sTokenPath                    = "/var/run/secrets/kubernetes.io/serviceaccount/token" //nolint:gosec // False positive G101: Potential hardcoded credentials. The detector only check if the token exists.
	k8sCertPath                     = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	authConfigmapNS                 = "kube-system"
	authConfigmapName               = "aws-auth"
	cwConfigmapNS                   = "amazon-cloudwatch"
	cwConfigmapName                 = "cluster-info"
	defaultCgroupPath               = "/proc/self/cgroup"
	containerIDLength               = 64
	configMapTimeout                = 10 * time.Second
	k8sAPITimeout                   = 10 * time.Second
	imdsTimeout                     = 2 * time.Second
	kubernetesServiceHostEnvVar     = "KUBERNETES_SERVICE_HOST"
	eksClusterStringIdentifier      = "-eks-"
	eksIRSATokenPathIdentifier      = "eks.amazonaws.com" //nolint:gosec // not a credential
	eksPodIdentityPathIdentifier    = "eks-pod-identity"
	awsWebIdentityTokenFileEnvVar   = "AWS_WEB_IDENTITY_TOKEN_FILE"            //nolint:gosec // env var name, not a credential
	awsContainerAuthTokenFileEnvVar = "AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE" //nolint:gosec // env var name, not a credential
	defaultNodeEnvVar               = "K8S_NODE_NAME"

	clusterNameAwsEksTag     = "aws:eks:cluster-name"
	clusterNameEksTag        = "eks:cluster-name"
	kubernetesClusterNameTag = "kubernetes.io/cluster/"
)

// detectorUtils is used for testing the resourceDetector by abstracting functions that rely on external systems.
type detectorUtils interface {
	fileExists(filename string) bool
	getConfigMap(ctx context.Context, namespace, name string) (map[string]string, error)
	getContainerID() (string, error)
	getServerVersion(ctx context.Context) (string, error)
	getNodeProviderID(ctx context.Context, nodeName string) (string, error)
}

// imdsClient captures the subset of EC2 IMDS methods needed by the detector.
type imdsClient interface {
	GetInstanceIdentityDocument(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error)
	GetMetadata(ctx context.Context, params *imds.GetMetadataInput, optFns ...func(*imds.Options)) (*imds.GetMetadataOutput, error)
}

// ec2Client captures the subset of EC2 API methods needed by the detector.
type ec2Client interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

// This struct will implement the detectorUtils interface.
type eksDetectorUtils struct {
	host   string
	client *http.Client
}

// configMap is the subset of a Kubernetes ConfigMap response needed by the detector.
type configMap struct {
	Data map[string]string `json:"data"`
}

// node is the subset of a Kubernetes Node response needed by the detector.
type node struct {
	Spec struct {
		ProviderID string `json:"providerID"`
	} `json:"spec"`
}

// serverVersion is the subset of a Kubernetes /version response needed by the detector.
type serverVersion struct {
	GitVersion string `json:"gitVersion"`
}

type nodeMetadata struct {
	instanceID       string
	availabilityZone string
	region           string
	accountID        string
	imageID          string
	instanceType     string
	hostname         string
	clusterName      string
}

type config struct {
	nodeEnvVar string
	filter     attribute.Filter
	awsConfig  *aws.Config
	imdsClient imdsClient
	ec2Client  ec2Client
}

// Option configures a [ResourceDetector].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithAttributeFilter sets a filter that controls which detected attributes
// are included in the returned resource. Only attributes for which filter
// returns true are included. By default all attributes are included.
func WithAttributeFilter(filter attribute.Filter) Option {
	return optionFunc(func(c *config) { c.filter = filter })
}

// WithNodeEnvVar sets the environment variable name from which the Kubernetes
// node name is read. Defaults to "K8S_NODE_NAME".
func WithNodeEnvVar(name string) Option {
	return optionFunc(func(c *config) { c.nodeEnvVar = name })
}

// WithAWSConfig sets the AWS SDK configuration used to query AWS APIs.
func WithAWSConfig(cfg aws.Config) Option {
	return optionFunc(func(c *config) { c.awsConfig = &cfg })
}

func withIMDSClient(client imdsClient) Option {
	return optionFunc(func(c *config) { c.imdsClient = client })
}

func withEC2Client(client ec2Client) Option {
	return optionFunc(func(c *config) { c.ec2Client = client })
}

// ResourceDetector for detecting resources running on Amazon EKS.
type ResourceDetector struct {
	cfg   config
	utils detectorUtils
	err   error
}

// Compile time assertion that ResourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*ResourceDetector)(nil)

// Compile time assertion that eksDetectorUtils implements the detectorUtils interface.
var _ detectorUtils = (*eksDetectorUtils)(nil)

// is this going to stop working with 1.20 when Docker is deprecated?
var containerIDRegex = regexp.MustCompile(`^.*/docker/(.+)$`)

// NewResourceDetector returns a resource detector that will detect AWS EKS resources.
func NewResourceDetector(opts ...Option) *ResourceDetector {
	cfg := config{nodeEnvVar: defaultNodeEnvVar}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	utils, err := newK8sDetectorUtils()
	return &ResourceDetector{cfg: cfg, utils: utils, err: err}
}

// Detect returns a Resource describing the Amazon EKS environment being run in.
func (detector *ResourceDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	if detector.err != nil {
		if errors.Is(detector.err, rest.ErrNotInCluster) {
			return resource.Empty(), nil
		}

		return nil, detector.err
	}

	if !isEKS(ctx, detector.utils) {
		return resource.Empty(), nil
	}

	// Create variable to hold resource attributes
	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSEKS,
	}

	nodeMeta := detector.detectNodeMetadata(ctx)

	if nodeMeta.accountID != "" {
		attributes = append(attributes, semconv.CloudAccountID(nodeMeta.accountID))
	}
	if nodeMeta.availabilityZone != "" {
		attributes = append(attributes, semconv.CloudAvailabilityZone(nodeMeta.availabilityZone))
	}
	if nodeMeta.region != "" {
		attributes = append(attributes, semconv.CloudRegion(nodeMeta.region))
	}
	if nodeMeta.instanceID != "" {
		attributes = append(attributes, semconv.HostID(nodeMeta.instanceID))
	}
	if nodeMeta.imageID != "" {
		attributes = append(attributes, semconv.HostImageID(nodeMeta.imageID))
	}
	if nodeMeta.instanceType != "" {
		attributes = append(attributes, semconv.HostType(nodeMeta.instanceType))
	}
	if nodeMeta.hostname != "" {
		attributes = append(attributes, semconv.HostNameKey.String(nodeMeta.hostname))
	}

	// Cluster name: prefer EC2 instance tags, fallback to CloudWatch ConfigMap.
	clusterName := nodeMeta.clusterName
	if clusterName == "" {
		var err error
		clusterName, err = getClusterName(ctx, detector.utils)
		if err != nil {
			return nil, err
		}
	}
	if clusterName != "" {
		attributes = append(attributes, semconv.K8SClusterName(clusterName))
	}

	// Get containerID and append to attributes
	containerID, err := detector.utils.getContainerID()
	if err != nil {
		return nil, err
	}
	if containerID != "" {
		attributes = append(attributes, semconv.ContainerID(containerID))
	}

	if detector.cfg.filter != nil {
		filtered := attributes[:0]
		for _, kv := range attributes {
			if detector.cfg.filter(kv) {
				filtered = append(filtered, kv)
			}
		}
		attributes = filtered
	}

	// Return new resource object with attributes
	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

func (detector *ResourceDetector) getAWSConfig(ctx context.Context) (aws.Config, error) {
	if detector.cfg.awsConfig != nil {
		return *detector.cfg.awsConfig, nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, err
	}
	cfg.Retryer = func() aws.Retryer {
		return retry.NewStandard(func(options *retry.StandardOptions) {
			options.MaxAttempts = 1
		})
	}
	return cfg, nil
}

func (detector *ResourceDetector) getIMDSClient(ctx context.Context) (imdsClient, error) {
	if detector.cfg.imdsClient != nil {
		return detector.cfg.imdsClient, nil
	}
	cfg, err := detector.getAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	return imds.NewFromConfig(cfg), nil
}

func (detector *ResourceDetector) getEC2Client(ctx context.Context, optFns ...func(*ec2.Options)) (ec2Client, error) {
	if detector.cfg.ec2Client != nil {
		return detector.cfg.ec2Client, nil
	}
	cfg, err := detector.getAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(cfg, optFns...), nil
}

func (detector *ResourceDetector) detectNodeMetadata(ctx context.Context) nodeMetadata {
	var meta nodeMetadata

	// 1. Try IMDS first with a bounded timeout.
	imdsCtx, cancel := context.WithTimeout(ctx, imdsTimeout)
	defer cancel()

	if imdsCl, err := detector.getIMDSClient(imdsCtx); err == nil {
		if docOut, err := imdsCl.GetInstanceIdentityDocument(imdsCtx, nil); err == nil && docOut != nil {
			meta.instanceID = docOut.InstanceID
			meta.availabilityZone = docOut.AvailabilityZone
			meta.region = docOut.Region
			meta.accountID = docOut.AccountID
			meta.imageID = docOut.ImageID
			meta.instanceType = docOut.InstanceType
			meta.hostname = getHostnameFromIMDS(imdsCtx, imdsCl)
		}
	}

	// 2. If instanceID is empty, fallback to K8s Node API (EKS Auto Mode / IMDS disabled).
	if meta.instanceID == "" {
		if nodeMeta, err := detector.getK8sNodeMetadata(ctx); err == nil {
			meta.instanceID = nodeMeta.instanceID
			meta.availabilityZone = nodeMeta.availabilityZone
			meta.region = nodeMeta.region
		}
	}

	// 3. If instanceID and region are available, query EC2 DescribeInstances for tags and missing fields.
	if meta.instanceID != "" && meta.region != "" {
		if ec2Cl, err := detector.getEC2Client(ctx, func(o *ec2.Options) { o.Region = meta.region }); err == nil {
			descOut, err := ec2Cl.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				InstanceIds: []string{meta.instanceID},
			})
			if err == nil && descOut != nil {
				meta.clusterName = getClusterNameTagFromReservations(descOut.Reservations)
				if len(descOut.Reservations) > 0 && len(descOut.Reservations[0].Instances) > 0 {
					inst := descOut.Reservations[0].Instances[0]
					if meta.accountID == "" {
						meta.accountID = aws.ToString(descOut.Reservations[0].OwnerId)
					}
					if meta.imageID == "" {
						meta.imageID = aws.ToString(inst.ImageId)
					}
					if meta.instanceType == "" {
						meta.instanceType = string(inst.InstanceType)
					}
					if meta.hostname == "" {
						meta.hostname = aws.ToString(inst.PrivateDnsName)
					}
				}
			}
		}
	}

	return meta
}

func (detector *ResourceDetector) getK8sNodeMetadata(ctx context.Context) (nodeMetadata, error) {
	nodeName := os.Getenv(detector.cfg.nodeEnvVar)
	if nodeName == "" {
		return nodeMetadata{}, errors.New("node name is empty")
	}

	providerID, err := detector.utils.getNodeProviderID(ctx, nodeName)
	if err != nil {
		return nodeMetadata{}, fmt.Errorf("failed to retrieve k8s node: %w", err)
	}

	region, availabilityZone, instanceID := parseRegionAndInstanceID(providerID)
	if region == "" || instanceID == "" {
		return nodeMetadata{}, fmt.Errorf("failed to parse region or instanceID from providerID: %s", providerID)
	}

	return nodeMetadata{
		region:           region,
		availabilityZone: availabilityZone,
		instanceID:       instanceID,
	}, nil
}

func parseRegionAndInstanceID(input string) (string, string, string) {
	// Format: "aws:///us-west-2a/i-049ca2df511bec762"
	parts := strings.Split(input, "/")
	if len(parts) < 5 || parts[0] != "aws:" || parts[3] == "" || parts[4] == "" {
		return "", "", ""
	}

	availabilityZone := parts[3]
	if len(availabilityZone) < 2 {
		return "", "", ""
	}
	region := availabilityZone[:len(availabilityZone)-1]
	instanceID := parts[4]

	return region, availabilityZone, instanceID
}

func getClusterNameTagFromReservations(reservations []types.Reservation) string {
	for _, reservation := range reservations {
		for i := range reservation.Instances {
			instance := reservation.Instances[i]
			for _, tag := range instance.Tags {
				key := aws.ToString(tag.Key)
				if key == "" {
					continue
				}
				if key == clusterNameAwsEksTag || key == clusterNameEksTag {
					return aws.ToString(tag.Value)
				}
				if after, ok := strings.CutPrefix(key, kubernetesClusterNameTag); ok {
					return after
				}
			}
		}
	}
	return ""
}

func getHostnameFromIMDS(ctx context.Context, client imdsClient) string {
	out, err := client.GetMetadata(ctx, &imds.GetMetadataInput{Path: "hostname"})
	if err != nil || out == nil || out.Content == nil {
		return ""
	}
	defer out.Content.Close()
	data, err := io.ReadAll(out.Content)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// isEKS checks if the current environment is running in EKS.
func isEKS(ctx context.Context, utils detectorUtils) bool {
	if !isK8s(utils) {
		return false
	}

	// Check for EKS-specific IRSA token file path
	if tokenFile := os.Getenv(awsWebIdentityTokenFileEnvVar); strings.Contains(tokenFile, eksIRSATokenPathIdentifier) {
		return true
	}

	// Check for EKS Pod Identity token file path
	if authFile := os.Getenv(awsContainerAuthTokenFileEnvVar); strings.Contains(authFile, eksPodIdentityPathIdentifier) {
		return true
	}

	// Check cluster server version for -eks- identifier
	if version, err := utils.getServerVersion(ctx); err == nil && strings.Contains(version, eksClusterStringIdentifier) {
		return true
	}

	// Fallback: check for aws-auth ConfigMap in kube-system
	awsAuth, err := utils.getConfigMap(ctx, authConfigmapNS, authConfigmapName)
	return err == nil && awsAuth != nil
}

// newK8sDetectorUtils creates utilities that fetch ConfigMaps over the in-cluster HTTP client.
func newK8sDetectorUtils() (*eksDetectorUtils, error) {
	// Get cluster configuration
	confs, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	client, err := rest.HTTPClientFor(confs)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client for Kubernetes: %w", err)
	}

	return &eksDetectorUtils{host: confs.Host, client: client}, nil
}

// isK8s checks if the current environment is running in a Kubernetes environment.
func isK8s(utils detectorUtils) bool {
	if os.Getenv(kubernetesServiceHostEnvVar) != "" {
		return true
	}
	return utils.fileExists(k8sTokenPath) && utils.fileExists(k8sCertPath)
}

// fileExists checks if a file with a given filename exists.
func (eksDetectorUtils) fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

// getConfigMap retrieves the configuration map from the k8s API.
func (eksUtils eksDetectorUtils) getConfigMap(ctx context.Context, namespace, name string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, configMapTimeout)
	defer cancel()

	u, err := url.JoinPath(eksUtils.host, "api", "v1", "namespaces", url.PathEscape(namespace), "configmaps", url.PathEscape(name))
	if err != nil {
		return nil, fmt.Errorf("failed to build ConfigMap URL for %s/%s: %w", namespace, name, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create ConfigMap request for %s/%s: %w", namespace, name, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := eksUtils.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ConfigMap %s/%s: %w", namespace, name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to retrieve ConfigMap %s/%s: unexpected status %s", namespace, name, resp.Status)
	}

	var cm configMap
	if err := json.NewDecoder(resp.Body).Decode(&cm); err != nil {
		return nil, fmt.Errorf("failed to decode ConfigMap %s/%s: %w", namespace, name, err)
	}

	return cm.Data, nil
}

// getServerVersion retrieves the Kubernetes cluster server version.
func (eksUtils eksDetectorUtils) getServerVersion(ctx context.Context) (string, error) {
	var v serverVersion
	if err := eksUtils.getK8sAPI(ctx, &v, "version"); err != nil {
		return "", fmt.Errorf("failed to retrieve server version: %w", err)
	}
	return v.GitVersion, nil
}

// getNodeProviderID retrieves the provider ID of the named Kubernetes node.
func (eksUtils eksDetectorUtils) getNodeProviderID(ctx context.Context, nodeName string) (string, error) {
	var n node
	if err := eksUtils.getK8sAPI(ctx, &n, "api", "v1", "nodes", url.PathEscape(nodeName)); err != nil {
		return "", fmt.Errorf("failed to retrieve node %s: %w", nodeName, err)
	}
	return n.Spec.ProviderID, nil
}

// getK8sAPI performs a GET request against the Kubernetes API and decodes the JSON response into out.
func (eksUtils eksDetectorUtils) getK8sAPI(ctx context.Context, out any, elem ...string) error {
	ctx, cancel := context.WithTimeout(ctx, k8sAPITimeout)
	defer cancel()

	u, err := url.JoinPath(eksUtils.host, elem...)
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := eksUtils.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

// getClusterName retrieves the clusterName resource attribute.
func getClusterName(ctx context.Context, utils detectorUtils) (string, error) {
	resp, err := utils.getConfigMap(ctx, cwConfigmapNS, cwConfigmapName)
	if err != nil {
		if isNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("getClusterName() error: %w", err)
	}

	return resp["cluster.name"], nil
}

func isNotFound(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "404") || strings.Contains(errStr, "not found")
}

// getContainerID returns the containerID if currently running within a container.
func (eksDetectorUtils) getContainerID() (string, error) {
	if runtime.GOOS != "linux" {
		return "", nil
	}

	fileData, err := os.ReadFile(defaultCgroupPath)
	if err != nil {
		return "", nil
	}

	// Retrieve containerID from file
	splitData := strings.SplitSeq(strings.TrimSpace(string(fileData)), "\n")
	for str := range splitData {
		if containerIDRegex.MatchString(str) {
			return str[len(str)-containerIDLength:], nil
		}
	}
	return "", nil
}
