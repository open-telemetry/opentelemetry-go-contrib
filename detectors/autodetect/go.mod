module go.opentelemetry.io/contrib/detectors/autodetect

go 1.25.0

require (
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/contrib/detectors/aws/ec2/v2 v2.5.1
	go.opentelemetry.io/contrib/detectors/aws/ecs v1.44.0
	go.opentelemetry.io/contrib/detectors/aws/eks v1.44.0
	go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk v0.16.0
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.69.0
	go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps v0.16.0
	go.opentelemetry.io/contrib/detectors/azure/azurevm v0.16.0
	go.opentelemetry.io/contrib/detectors/docker v0.16.0
	go.opentelemetry.io/contrib/detectors/gcp v1.44.0
	go.opentelemetry.io/contrib/detectors/hetzner v0.16.0
	go.opentelemetry.io/contrib/detectors/k8sapi v0.16.0
	go.opentelemetry.io/otel v1.44.1-0.20260717185620-3f1e0cf6869a
	go.opentelemetry.io/otel/sdk v1.44.1-0.20260625150014-c84013202f01
)

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.34.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/aws/aws-sdk-go-v2 v1.43.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.31 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.30 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.31 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.32 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.5.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.33.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.38.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.45.0 // indirect
	github.com/aws/smithy-go v1.27.4 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/brunoscheufler/aws-ecs-metadata-go v0.0.0-20221221133751-67e37ae746cd // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v1.0.0 // indirect
	github.com/go-openapi/jsonreference v1.0.0 // indirect
	github.com/go-openapi/swag v0.27.3 // indirect
	github.com/go-openapi/swag/cmdutils v0.27.3 // indirect
	github.com/go-openapi/swag/conv v0.27.3 // indirect
	github.com/go-openapi/swag/fileutils v0.27.3 // indirect
	github.com/go-openapi/swag/jsonutils v0.27.3 // indirect
	github.com/go-openapi/swag/loading v0.27.3 // indirect
	github.com/go-openapi/swag/mangling v0.27.3 // indirect
	github.com/go-openapi/swag/netutils v0.27.3 // indirect
	github.com/go-openapi/swag/pools v0.27.3 // indirect
	github.com/go-openapi/swag/stringutils v0.27.3 // indirect
	github.com/go-openapi/swag/typeutils v0.27.3 // indirect
	github.com/go-openapi/swag/yamlutils v0.27.3 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hetznercloud/hcloud-go/v2 v2.45.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/moby/api v1.54.2 // indirect
	github.com/moby/moby/client v0.4.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.24.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.1-0.20260625150014-c84013202f01 // indirect
	go.opentelemetry.io/otel/trace v1.44.1-0.20260625150014-c84013202f01 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.35.4 // indirect
	k8s.io/apimachinery v0.35.4 // indirect
	k8s.io/client-go v0.35.4 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-openapi v0.0.0-20260721132016-d427ff9ee9ad // indirect
	k8s.io/utils v0.0.0-20260707023825-cf1189d6abe3 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.4.2 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

replace go.opentelemetry.io/contrib/detectors/azure/azurecontainerapps => ../azure/azurecontainerapps

replace go.opentelemetry.io/contrib/detectors/azure/azurevm => ../azure/azurevm

replace go.opentelemetry.io/contrib/detectors/aws/lambda => ../aws/lambda

replace go.opentelemetry.io/contrib/detectors/aws/eks => ../aws/eks

replace go.opentelemetry.io/contrib/detectors/aws/ecs => ../aws/ecs

replace go.opentelemetry.io/contrib/detectors/aws/elasticbeanstalk => ../aws/elasticbeanstalk

replace go.opentelemetry.io/contrib/detectors/gcp => ../gcp

replace go.opentelemetry.io/contrib/detectors/hetzner => ../hetzner

replace go.opentelemetry.io/contrib/detectors/k8sapi => ../k8sapi

replace go.opentelemetry.io/contrib/detectors/aws/ec2/v2 => ../aws/ec2/v2

replace go.opentelemetry.io/contrib/detectors/docker => ../docker
