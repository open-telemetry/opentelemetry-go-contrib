// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package autodetect provides functionality to configures and use a set of
// resource detectors at runtime.
//
// It registers every resource detector shipped in this repository, so
// importing it pulls every detector module into the build. To keep the
// detectors with third-party dependencies out of the module graph, use
// [go.opentelemetry.io/contrib/detectors/autodetect/registry] instead, and
// read the registration ownership model documented there before registering
// an ID.
package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/akamai"
	alibabaecs "go.opentelemetry.io/contrib/detectors/alibaba/ecs"
	"go.opentelemetry.io/contrib/detectors/autodetect/registry"
	"go.opentelemetry.io/contrib/detectors/aws/ec2/v2"
	"go.opentelemetry.io/contrib/detectors/aws/ecs"
	"go.opentelemetry.io/contrib/detectors/aws/eks"
	"go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk"
	"go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/detectors/azure/azureaks"
	"go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps"
	"go.opentelemetry.io/contrib/detectors/azure/azurevm"
	"go.opentelemetry.io/contrib/detectors/consul"
	"go.opentelemetry.io/contrib/detectors/docker"
	"go.opentelemetry.io/contrib/detectors/dynatrace"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/detectors/hetzner"
	"go.opentelemetry.io/contrib/detectors/ibmcloud/classic"
	"go.opentelemetry.io/contrib/detectors/ibmcloud/vpc"
	"go.opentelemetry.io/contrib/detectors/k8sapi"
	"go.opentelemetry.io/contrib/detectors/kubeadm"
	"go.opentelemetry.io/contrib/detectors/openshift"
	"go.opentelemetry.io/contrib/detectors/openstack/nova"
	"go.opentelemetry.io/contrib/detectors/scaleway"
	"go.opentelemetry.io/contrib/detectors/vultr"
)

// ID represents the unique identifier of a resource detector.
type ID = registry.ID

// ErrUnknownDetector is returned when a detector is not registered.
var ErrUnknownDetector = registry.ErrUnknownDetector

// Register registers a new resource detector function with the given ID.
func Register(id ID, fn func() resource.Detector) {
	registry.Register(id, fn)
}

// Registered returns a sorted slice of all registered resource detector IDs.
func Registered() []ID {
	return registry.Registered()
}

// Detector returns a [resource.Detector] composed of the detectors
// identified by the provided IDs. If an ID is not recognized,
// ErrUnknownDetector is returned. The returned detector merges all the
// resource from each detector when Detect is called. The order of the merge is
// not guaranteed.
func Detector(ids ...ID) (resource.Detector, error) {
	return registry.Detector(ids...)
}

// The detectors that need nothing beyond the SDK are registered by
// [go.opentelemetry.io/contrib/detectors/autodetect/registry].
var (
	// IDHost is the ID for the host detector. This detector detects the
	// "host.name" attribute from the os.Hostname function.
	IDHost = registry.IDHost
	// IDHostID is the ID for the host ID detector. This detector detects the
	// "host.id" attribute, which is a unique identifier for the host (e.g.,
	// machine-id, UUID).
	IDHostID = registry.IDHostID
	// IDTelemetrySDK is the ID for the telemetry SDK detector. This detector
	// detects the "telemetry.sdk.name", "telemetry.sdk.language", and
	// "telemetry.sdk.version" attributes, which provide information about the
	// SDK being used.
	IDTelemetrySDK = registry.IDTelemetrySDK
	// IDOSType is the ID for the OS type detector. This detector detects the
	// "os.type" attribute, which indicates the type of operating system (e.g.,
	// "linux", "windows", "darwin").
	IDOSType = registry.IDOSType
	// IDOSDescription is the ID for the OS description detector. This detector
	// detects the "os.description" attribute, which provides a human-readable
	// description of the operating system.
	IDOSDescription = registry.IDOSDescription
	// IDProcessPID is the ID for the process PID detector. This detector
	// detects the "process.pid" attribute, which is the process ID of the
	// current process.
	IDProcessPID = registry.IDProcessPID
	// IDProcessExecutableName is the ID for the process executable name
	// detector. This detector detects the "process.executable.name" attribute,
	// which is the name of the executable file for the current process.
	IDProcessExecutableName = registry.IDProcessExecutableName
	// IDProcessExecutablePath is the ID for the process executable path
	// detector. This detector detects the "process.executable.path" attribute,
	// which is the full path to the executable file for the current process.
	IDProcessExecutablePath = registry.IDProcessExecutablePath
	// IDProcessCommandArgs is the ID for the process command arguments
	// detector. This detector detects the "process.command.args" attribute,
	// which is the command line arguments used to start the current process.
	//
	// Warning! This detector will include process command line arguments. If
	// these contain sensitive information it will be included in the exported
	// resource.
	IDProcessCommandArgs = registry.IDProcessCommandArgs
	// IDProcessOwner is the ID for the process owner detector. This detector
	// detects the "process.owner" attribute, which is the user who owns the
	// current process.
	IDProcessOwner = registry.IDProcessOwner
	// IDProcessRuntimeName is the ID for the process runtime name detector.
	// This detector detects the "process.runtime.name" attribute, which is the
	// name of the runtime environment for the current process (e.g., "go",
	// "python", "java").
	IDProcessRuntimeName = registry.IDProcessRuntimeName
	// IDProcessRuntimeVersion is the ID for the process runtime version
	// detector. This detector detects the "process.runtime.version" attribute,
	// which is the version of the runtime environment for the current process
	// (e.g., "1.16.3", "3.8.5").
	IDProcessRuntimeVersion = registry.IDProcessRuntimeVersion
	// IDProcessRuntimeDescription is the ID for the process runtime
	// description detector. This detector detects the
	// "process.runtime.description" attribute, which provides an additional
	// description of the runtime environment for the current process (e.g.,
	// "Go runtime version 1.16.3", "Python 3.8.5").
	IDProcessRuntimeDescription = registry.IDProcessRuntimeDescription
	// IDContainer is the ID for the container detector. This detector detects
	// the "container.id" attribute, which is a unique identifier for the
	// container in which the process is running. This is useful for
	// identifying the container in which the process is running, especially in
	// containerized environments like Kubernetes or Docker.
	IDContainer = registry.IDContainer
)

var (
	// IDAkamai is the ID for the Akamai detector that detects resource
	// attributes on Akamai Connected Cloud compute instances (see
	// akamai.NewResourceDetector for details).
	IDAkamai = registry.ID("akamai")
	// IDAlibabaECS is the ID for the Alibaba Cloud ECS detector that detects
	// resource attributes on Alibaba Cloud Elastic Compute Service (ECS)
	// instances (see alibabaecs.NewResourceDetector for details).
	IDAlibabaECS = registry.ID("alibaba.ecs")
	// IDAWSEC2 is the ID for the AWS EC2 detector that detects resource
	// attributes on Amazon Web Services (AWS) EC2 instances (see
	// ec2.NewResourceDetector for details).
	IDAWSEC2 = registry.ID("aws.ec2")
	// IDAWSECS is the ID for the AWS ECS detector that detects resource
	// attributes on Amazon Web Services (AWS) ECS clusters (see
	// ecs.NewResourceDetector for details).
	IDAWSECS = registry.ID("aws.ecs")
	// IDAWSEKS is the ID for the AWS EKS detector that detects resource
	// attributes on Amazon Web Services (AWS) EKS clusters (see
	// eks.NewResourceDetector for details).
	IDAWSEKS = registry.ID("aws.eks")
	// IDAWSLambda is the ID for the AWS Lambda detector that detects resource
	// attributes on Amazon Web Services (AWS) Lambda functions (see
	// lambda.NewResourceDetector for details).
	IDAWSLambda = registry.ID("aws.lambda")
	// IDAWSElasticBeanstalk is the ID for the AWS Elastic Beanstalk detector that detects resource
	// attributes on Amazon Web Services (AWS) Elastic Beanstalk (see
	// elasticbeanstalk.NewResourceDetector for details).
	IDAWSElasticBeanstalk = registry.ID("aws.elasticbeanstalk")
	// IDAzureAKS is the ID for the Azure Kubernetes Service detector that
	// detects resource attributes on Microsoft Azure Kubernetes Service (see
	// azureaks.NewResourceDetector for details).
	IDAzureAKS = registry.ID("azure.aks")
	// IDAzureContainerApps is the ID for the Azure Container Apps detector
	// that detects resource attributes on Microsoft Azure Container Apps (see
	// azurecontainerapps.NewResourceDetector for details).
	IDAzureContainerApps = registry.ID("azure.container_apps")
	// IDAzureVM is the ID for the Azure VM detector that detects resource
	// attributes on Microsoft Azure virtual machines (see azurevm.New for
	// details).
	IDAzureVM = registry.ID("azure.vm")
	// IDConsul is the ID for the Consul detector that detects resource
	// attributes from a Consul agent (see consul.NewResourceDetector for
	// details).
	IDConsul = registry.ID("consul")
	// IDDynatrace is the ID for the Dynatrace detector that detects resource
	// attributes from the enrichment files written by the Dynatrace OneAgent
	// (see dynatrace.NewResourceDetector for details).
	IDDynatrace = registry.ID("dynatrace")
	// IDDocker is the ID for the Docker detector that detects resource
	// attributes on Docker containers (see docker.NewResourceDetector for
	// details).
	IDDocker = registry.ID("docker")
	// IDGCP is the ID for the GCP detector that detects resource attributes on
	// Google Cloud Platform (GCP) environments (see gcp.NewDetector for
	// details).
	IDGCP = registry.ID("gcp")
	// IDHetzner is the ID for the Hetzner Cloud detector that detects resource
	// attributes on Hetzner Cloud servers (see hetzner.NewResourceDetector for
	// details).
	IDHetzner = registry.ID("hetzner")
	// IDIBMCloudClassic is the ID for the IBM Cloud Classic detector that
	// detects resource attributes on IBM Cloud Classic Infrastructure
	// (SoftLayer) instances (see classic.NewResourceDetector for details).
	IDIBMCloudClassic = registry.ID("ibmcloud.classic")
	// IDIBMCloudVPC is the ID for the IBM Cloud VPC detector that detects
	// resource attributes on IBM Cloud VPC virtual server instances (see
	// vpc.NewResourceDetector for details).
	IDIBMCloudVPC = registry.ID("ibmcloud.vpc")
	// IDOpenStackNova is the ID for the OpenStack Nova detector that detects
	// resource attributes on OpenStack Nova compute instances (see
	// nova.NewResourceDetector for details).
	IDOpenStackNova = registry.ID("openstack.nova")
	// IDK8sAPI is the ID for the Kubernetes API detector that detects resource
	// attributes from the Kubernetes API (see k8sapi.NewResourceDetector for
	// details).
	IDK8sAPI = registry.ID("k8sapi")
	// IDKubeadm is the ID for the kubeadm detector that detects resource
	// attributes of the kubeadm-provisioned Kubernetes cluster the process is
	// running in (see kubeadm.NewResourceDetector for details).
	IDKubeadm = registry.ID("kubeadm")
	// IDOpenShift is the ID for the OpenShift detector that detects resource
	// attributes of OpenShift 4 clusters (see openshift.NewResourceDetector
	// for details).
	IDOpenShift = registry.ID("openshift")
	// IDScaleway is the ID for the Scaleway detector that detects resource
	// attributes on Scaleway Instances (see scaleway.NewResourceDetector for
	// details).
	IDScaleway = registry.ID("scaleway")
	// IDVultr is the ID for the Vultr detector that detects resource attributes
	// on Vultr Cloud Compute instances (see vultr.NewResourceDetector for
	// details).
	IDVultr = registry.ID("vultr")
)

func init() {
	registry.Register(IDAkamai, func() resource.Detector { return akamai.NewResourceDetector() })
	registry.Register(IDAlibabaECS, func() resource.Detector { return alibabaecs.NewResourceDetector() })
	registry.Register(IDAWSEC2, ec2.NewResourceDetector)
	registry.Register(IDAWSECS, ecs.NewResourceDetector)
	registry.Register(IDAWSEKS, eks.NewResourceDetector)
	registry.Register(IDAWSLambda, func() resource.Detector { return lambda.NewResourceDetector() })
	registry.Register(IDAWSElasticBeanstalk, func() resource.Detector { return elasticbeanstalk.NewResourceDetector() })
	registry.Register(IDAzureAKS, func() resource.Detector { return azureaks.NewResourceDetector() })
	registry.Register(IDAzureContainerApps, func() resource.Detector { return azurecontainerapps.NewResourceDetector() })
	registry.Register(IDAzureVM, func() resource.Detector { return azurevm.New() })
	registry.Register(IDConsul, func() resource.Detector { return consul.NewResourceDetector() })
	registry.Register(IDDynatrace, func() resource.Detector { return dynatrace.NewResourceDetector() })
	registry.Register(IDDocker, func() resource.Detector { return docker.NewResourceDetector() })
	registry.Register(IDGCP, gcp.NewDetector)
	registry.Register(IDHetzner, func() resource.Detector { return hetzner.NewResourceDetector() })
	registry.Register(IDIBMCloudClassic, func() resource.Detector { return classic.NewResourceDetector() })
	registry.Register(IDOpenStackNova, func() resource.Detector { return nova.NewResourceDetector() })
	registry.Register(IDIBMCloudVPC, func() resource.Detector { return vpc.NewResourceDetector() })
	registry.Register(IDK8sAPI, func() resource.Detector { return k8sapi.NewResourceDetector() })
	registry.Register(IDKubeadm, func() resource.Detector { return kubeadm.NewResourceDetector() })
	registry.Register(IDOpenShift, func() resource.Detector { return openshift.NewResourceDetector() })
	registry.Register(IDScaleway, func() resource.Detector { return scaleway.NewResourceDetector() })
	registry.Register(IDVultr, func() resource.Detector { return vultr.NewResourceDetector() })
}
