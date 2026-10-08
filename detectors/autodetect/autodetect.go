// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package autodetect provides functionality to configures and use a set of
// resource detectors at runtime.
//
// It registers every resource detector shipped in this repository. To register
// only some of them, and keep the dependencies of the others out of the module
// graph, use [go.opentelemetry.io/contrib/detectors/autodetect/registry]
// instead and call [registry.Register] with the detectors the build needs.
package autodetect

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/autodetect/registry"
	"go.opentelemetry.io/contrib/detectors/aws/ec2/v2"
	"go.opentelemetry.io/contrib/detectors/aws/ecs"
	"go.opentelemetry.io/contrib/detectors/aws/eks"
	"go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk"
	"go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps"
	"go.opentelemetry.io/contrib/detectors/azure/azurevm"
	"go.opentelemetry.io/contrib/detectors/docker"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/detectors/hetzner"
	"go.opentelemetry.io/contrib/detectors/ibmcloud/vpc"
	"go.opentelemetry.io/contrib/detectors/k8sapi"
	"go.opentelemetry.io/contrib/detectors/kubeadm"
	"go.opentelemetry.io/contrib/detectors/openshift"
	"go.opentelemetry.io/contrib/detectors/vultr"
)

// ID represents the unique identifier of a resource detector.
type ID = registry.ID

// ErrUnknownDetector is returned when a detector is not registered.
var ErrUnknownDetector = registry.ErrUnknownDetector

// These functions are provided by
// [go.opentelemetry.io/contrib/detectors/autodetect/registry].
var (
	Register   = registry.Register
	Registered = registry.Registered
	Detector   = registry.Detector
)

// The detectors that need nothing beyond the SDK are registered by
// [go.opentelemetry.io/contrib/detectors/autodetect/registry].
var (
	IDHost                      = registry.IDHost
	IDHostID                    = registry.IDHostID
	IDTelemetrySDK              = registry.IDTelemetrySDK
	IDOSType                    = registry.IDOSType
	IDOSDescription             = registry.IDOSDescription
	IDProcessPID                = registry.IDProcessPID
	IDProcessExecutableName     = registry.IDProcessExecutableName
	IDProcessExecutablePath     = registry.IDProcessExecutablePath
	IDProcessCommandArgs        = registry.IDProcessCommandArgs
	IDProcessOwner              = registry.IDProcessOwner
	IDProcessRuntimeName        = registry.IDProcessRuntimeName
	IDProcessRuntimeVersion     = registry.IDProcessRuntimeVersion
	IDProcessRuntimeDescription = registry.IDProcessRuntimeDescription
	IDContainer                 = registry.IDContainer
)

var ( // IDAWSEC2 is the ID for the AWS EC2 detector that detects resource
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
	// IDAzureContainerApps is the ID for the Azure Container Apps detector
	// that detects resource attributes on Microsoft Azure Container Apps (see
	// azurecontainerapps.NewResourceDetector for details).
	IDAzureContainerApps = registry.ID("azure.container_apps")
	// IDAzureVM is the ID for the Azure VM detector that detects resource
	// attributes on Microsoft Azure virtual machines (see azurevm.New for
	// details).
	IDAzureVM = registry.ID("azure.vm")
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
	// IDIBMCloudVPC is the ID for the IBM Cloud VPC detector that detects
	// resource attributes on IBM Cloud VPC virtual server instances (see
	// vpc.NewResourceDetector for details).
	IDIBMCloudVPC = registry.ID("ibmcloud.vpc")
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
	// IDVultr is the ID for the Vultr detector that detects resource attributes
	// on Vultr Cloud Compute instances (see vultr.NewResourceDetector for
	// details).
	IDVultr = registry.ID("vultr")
)

func init() {
	registry.Register(IDAWSEC2, ec2.NewResourceDetector)
	registry.Register(IDAWSECS, ecs.NewResourceDetector)
	registry.Register(IDAWSEKS, eks.NewResourceDetector)
	registry.Register(IDAWSLambda, lambda.NewResourceDetector)
	registry.Register(IDAWSElasticBeanstalk, func() resource.Detector { return elasticbeanstalk.NewResourceDetector() })
	registry.Register(IDAzureContainerApps, func() resource.Detector { return azurecontainerapps.NewResourceDetector() })
	registry.Register(IDAzureVM, func() resource.Detector { return azurevm.New() })
	registry.Register(IDDocker, func() resource.Detector { return docker.NewResourceDetector() })
	registry.Register(IDGCP, gcp.NewDetector)
	registry.Register(IDHetzner, func() resource.Detector { return hetzner.NewResourceDetector() })
	registry.Register(IDIBMCloudVPC, func() resource.Detector { return vpc.NewResourceDetector() })
	registry.Register(IDK8sAPI, func() resource.Detector { return k8sapi.NewResourceDetector() })
	registry.Register(IDKubeadm, func() resource.Detector { return kubeadm.NewResourceDetector() })
	registry.Register(IDOpenShift, func() resource.Detector { return openshift.NewResourceDetector() })
	registry.Register(IDVultr, func() resource.Detector { return vultr.NewResourceDetector() })
}
