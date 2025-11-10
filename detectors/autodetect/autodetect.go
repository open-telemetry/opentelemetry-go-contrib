// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package autodetect provides functionality to configures and use a set of
// resource detectors at runtime.
package autodetect // import "go.opentelemetry.io/contrib/detectors/autodetect"

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/aws/ec2/v2"
	"go.opentelemetry.io/contrib/detectors/aws/ecs"
	"go.opentelemetry.io/contrib/detectors/aws/eks"
	"go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/detectors/azure/azurevm"
	"go.opentelemetry.io/contrib/detectors/gcp"
)

var (
	// IDAWSEC2 is the ID for the AWS EC2 detector that detects resource
	// attributes on Amazon Web Services (AWS) EC2 instances (see
	// ec2.NewResourceDetector for details).
	IDAWSEC2 = ID("aws.ec2")
	// IDAWSECS is the ID for the AWS ECS detector that detects resource
	// attributes on Amazon Web Services (AWS) ECS clusters (see
	// ecs.NewResourceDetector for details).
	IDAWSECS = ID("aws.ecs")
	// IDAWSEKS is the ID for the AWS EKS detector that detects resource
	// attributes on Amazon Web Services (AWS) EKS clusters (see
	// eks.NewResourceDetector for details).
	IDAWSEKS = ID("aws.eks")
	// IDAWSLambda is the ID for the AWS Lambda detector that detects resource
	// attributes on Amazon Web Services (AWS) Lambda functions (see
	// lambda.NewResourceDetector for details).
	IDAWSLambda = ID("aws.lambda")
	// IDAzureVM is the ID for the Azure VM detector that detects resource
	// attributes on Microsoft Azure virtual machines (see azurevm.New for
	// details).
	IDAzureVM = ID("azure.vm")
	// IDGCP is the ID for the GCP detector that detects resource attributes on
	// Google Cloud Platform (GCP) environments (see gcp.NewDetector for
	// details).
	IDGCP = ID("gcp")
	// IDHost is the ID for the host detector. This detector detects the
	// "host.name" attribute from the os.Hostname function.
	IDHost = ID("host")
	// IDHostID is the ID for the host ID detector. This detector detects the
	// "host.id" attribute, which is a unique identifier for the host (e.g.,
	// machine-id, UUID).
	IDHostID = ID("host.id")
	// IDTelemetrySDK is the ID for the telemetry SDK detector. This detector
	// detects the "telemetry.sdk.name", "telemetry.sdk.language", and
	// "telemetry.sdk.version" attributes, which provide information about the
	// SDK being used.
	IDTelemetrySDK = ID("telemetry.sdk")
	// IDOSType is the ID for the OS type detector. This detector detects the
	// "os.type" attribute, which indicates the type of operating system (e.g.,
	// "linux", "windows", "darwin").
	IDOSType = ID("os.type")
	// IDOSDescription is the ID for the OS description detector. This detector
	// detects the "os.description" attribute, which provides a human-readable
	// description of the operating system.
	IDOSDescription = ID("os.description")
	// IDProcessPID is the ID for the process PID detector. This detector
	// detects the "process.pid" attribute, which is the process ID of the
	// current process.
	IDProcessPID = ID("process.pid")
	// IDProcessExecutableName is the ID for the process executable name
	// detector. This detector detects the "process.executable.name" attribute,
	// which is the name of the executable file for the current process.
	IDProcessExecutableName = ID("process.executable.name")
	// IDProcessExecutablePath is the ID for the process executable path
	// detector. This detector detects the "process.executable.path" attribute,
	// which is the full path to the executable file for the current process.
	IDProcessExecutablePath = ID("process.executable.path")
	// IDProcessCommandArgs is the ID for the process command arguments
	// detector. This detector detects the "process.command.args" attribute,
	// which is the command line arguments used to start the current process.
	//
	// Warning! This detector will include process command line arguments. If
	// these contain sensitive information it will be included in the exported
	// resource.
	IDProcessCommandArgs = ID("process.command.args")
	// IDProcessOwner is the ID for the process owner detector. This detector
	// detects the "process.owner" attribute, which is the user who owns the
	// current process.
	IDProcessOwner = ID("process.owner")
	// IDProcessRuntimeName is the ID for the process runtime name detector.
	// This detector detects the "process.runtime.name" attribute, which is the
	// name of the runtime environment for the current process (e.g., "go",
	// "python", "java").
	IDProcessRuntimeName = ID("process.runtime.name")
	// IDProcessRuntimeVersion is the ID for the process runtime version
	// detector. This detector detects the "process.runtime.version" attribute,
	// which is the version of the runtime environment for the current process
	// (e.g., "1.16.3", "3.8.5").
	IDProcessRuntimeVersion = ID("process.runtime.version")
	// IDProcessRuntimeDescription is the ID for the process runtime
	// description detector. This detector detects the
	// "process.runtime.description" attribute, which provides an additional
	// description of the runtime environment for the current process (e.g.,
	// "Go runtime version 1.16.3", "Python 3.8.5").
	IDProcessRuntimeDescription = ID("process.runtime.description")
	// IDContainer is the ID for the container detector. This detector detects
	// the "container.id" attribute, which is a unique identifier for the
	// container in which the process is running. This is useful for
	// identifying the container in which the process is running, especially in
	// containerized environments like Kubernetes or Docker.
	IDContainer = ID("container")
)

var (
	registryMu sync.Mutex
	registry   = map[ID]func() resource.Detector{
		IDAWSEC2:    ec2.NewResourceDetector,
		IDAWSECS:    ecs.NewResourceDetector,
		IDAWSEKS:    eks.NewResourceDetector,
		IDAWSLambda: lambda.NewResourceDetector,

		IDAzureVM: func() resource.Detector {
			return azurevm.New()
		},

		IDGCP: gcp.NewDetector,

		IDHost:   optFactory(resource.WithHost()),
		IDHostID: optFactory(resource.WithHostID()),

		IDTelemetrySDK: optFactory(resource.WithTelemetrySDK()),

		IDOSType:        optFactory(resource.WithOSType()),
		IDOSDescription: optFactory(resource.WithOSDescription()),

		IDProcessPID:                optFactory(resource.WithProcessPID()),
		IDProcessExecutableName:     optFactory(resource.WithProcessExecutableName()),
		IDProcessExecutablePath:     optFactory(resource.WithProcessExecutablePath()),
		IDProcessCommandArgs:        optFactory(resource.WithProcessCommandArgs()),
		IDProcessOwner:              optFactory(resource.WithProcessOwner()),
		IDProcessRuntimeName:        optFactory(resource.WithProcessRuntimeName()),
		IDProcessRuntimeVersion:     optFactory(resource.WithProcessRuntimeVersion()),
		IDProcessRuntimeDescription: optFactory(resource.WithProcessRuntimeDescription()),

		IDContainer: optFactory(resource.WithContainer()),
	}
)

// ID represents the unique identifier of a resource detector.
type ID string

// Register registers a new resource detector function with the given ID.
func Register(id ID, fn func() resource.Detector) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[id]; exists {
		panic("detector already registered: " + id)
	}
	registry[id] = fn
}

// Registered returns a sorted slice of all registered resource detector IDs.
func Registered() []ID {
	registryMu.Lock()
	defer registryMu.Unlock()
	out := make([]ID, 0, len(registry))
	for id := range registry {
		out = append(out, id)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

// optDetector is a resource.Detector that uses a resource.Option to
// create a resource.Resource. This is useful for detectors that
// do not require any additional logic beyond creating a resource
// from a resource.Option but do not export a concrete resource.Detector type
// directly.
type optDetector struct {
	opt resource.Option
}

var _ resource.Detector = optDetector{}

// optFactory returns a function that creates an resource.Detector factory
// function with the given resource.Option.
func optFactory(opt resource.Option) func() resource.Detector {
	return func() resource.Detector { return optDetector{opt: opt} }
}

// Detect returns the resource.Resource created by the resource.Option passed
// to the optDetector.
func (d optDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx, d.opt)
}

// composite is a [resource.Detector] that composes multiple
// [resource.Detector] into a single instance.
type composite struct {
	detectors []resource.Detector
}

var _ resource.Detector = &composite{}

// newComposite returns a new composite detector that runs the provided
// detectors in parallel and merges their results.
func newComposite(detectors []resource.Detector) *composite {
	return &composite{detectors: detectors}
}

// Detect runs all the detectors in parallel and merges the results into a
// single resource.Resource. If any detector returns an error, it is
// collected and returned as a single error. The resulting
// resource.Resource is the merge of all the resources returned by the
// detectors. If there is a merge conflict (e.g., different schema URLs),
// the resulting resource.Resource will be a partial resource with an
// error indicating the conflict (see
// [resource.ErrSchemaURLConflict] for more information).
func (c *composite) Detect(ctx context.Context) (*resource.Resource, error) {
	out := <-mergeDetections(doDetect(ctx, c.detectors))
	return out.res, out.err
}

// detection is the result of a [resource.Detector] detection.
type detection struct {
	res *resource.Resource
	err error
}

// doDetect runs all the detectors concurrently in their own goroutines. All
// detections are sent on the returned channel, and the channel is closed once
// all detections are complete.
func doDetect(ctx context.Context, detectors []resource.Detector) <-chan detection {
	detected := make(chan detection, len(detectors))
	go func() {
		var wg sync.WaitGroup
		for _, detector := range detectors {
			wg.Add(1)
			go func(d resource.Detector) {
				defer wg.Done()
				r, e := d.Detect(ctx)
				detected <- detection{res: r, err: e}
			}(detector)
		}

		wg.Wait()
		close(detected)
	}()
	return detected
}

// mergeDetections merges the results of multiple detections received on the in
// chan into a single detection result. The resulting detection is sent on
// the returned channel. If any of the detections have an error, it is
// collected and returned as a single error. The resulting resource.Resource
// is the merge of all the resources returned by the detectors. If there is a
// merge conflict (e.g., different schema URLs), the resulting
// resource.Resource will be a partial resource with an
// error indicating the conflict (see
// [resource.ErrSchemaURLConflict] for more information).
func mergeDetections(in <-chan detection) <-chan detection {
	merged := make(chan detection, 1)
	go func() {
		m := detection{res: resource.Empty()}
		for d := range in {
			m.err = errors.Join(m.err, d.err)

			var err error
			m.res, err = resource.Merge(m.res, d.res)
			if err != nil {
				// Merge errors are not recoverable.
				m.res, m.err = nil, err
				break
			}
		}
		merged <- m
		close(merged)
	}()
	return merged
}

// ErrUnknownDetector is returned when an unknown resource detector ID is
// requested.
var ErrUnknownDetector = errors.New("unknown resource detector")

// Detector returns a [resource.Detector] composed of the detectors
// identified by the provided IDs. If an ID is not recognized,
// ErrUnknownDetector is returned. The returned detector merges all the
// resource from each detector when Detect is called. The order of the merge is
// not guaranteed.
func Detector(ids ...ID) (resource.Detector, error) {
	registryMu.Lock()
	defer registryMu.Unlock()

	var (
		detectors []resource.Detector
		err       error
	)

	for _, id := range ids {
		fn, exists := registry[id]
		if !exists {
			e := fmt.Errorf("%w: %s", ErrUnknownDetector, id)
			err = errors.Join(err, e)
			continue
		}
		detectors = append(detectors, fn())
	}
	return newComposite(detectors), err
}
