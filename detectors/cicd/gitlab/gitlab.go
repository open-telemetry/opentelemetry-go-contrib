package gitlab

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"os"
)

const (
	gitlabCIEnvVar           = "GITLAB_CI"
	gitlabPipelineNameEnvVar = "CI_PIPELINE_NAME"
	gitlabPipelineIdEnvVar   = "CI_PIPELINE_ID"
	gitlabJobIdEnvVar        = "CI_JOB_ID"
	gitlabJobNameEnvVar      = "CI_JOB_NAME"
	gitlabJobStageEnvVar     = "CI_JOB_STAGE"
	gitlabJobUrlEnvVar       = "CI_JOB_URL"

	gitlabCommitRefNameEnvVar   = "CI_COMMIT_REF_NAME"
	gitlabCommitTagEnvVar       = "CI_COMMIT_TAG"
	gitlabMergeRequestIIDEnvVar = "CI_MERGE_REQUEST_IID"

	gitlabProjectUrlEnvVar = "CI_PROJECT_URL"
	gitlabProjectIDEnvVar  = "CI_PROJECT_ID"
)

type resourceDetector struct {
}

// compile time assertion that resourceDetector implements the resource.Detector interface.
var _ resource.Detector = (*resourceDetector)(nil)

// NewResourceDetector returns a [ResourceDetector] that will detect Gitlab Pipeline resources.
func NewResourceDetector() resource.Detector {
	return &resourceDetector{}
}

func (detector *resourceDetector) Detect(_ context.Context) (*resource.Resource, error) {

	var attributes []attribute.KeyValue

	isGitlabCI := os.Getenv(gitlabCIEnvVar) == "true"

	if isGitlabCI {
		attributes = append(attributes, detectCICDAttributes()...)
		attributes = append(attributes, detectVCSAttributes()...)
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

// detectCICDAttributes https://github.com/open-telemetry/semantic-conventions/blob/main/docs/attributes-registry/cicd.md
func detectCICDAttributes() []attribute.KeyValue {
	var attributes []attribute.KeyValue

	ciPipelineName := os.Getenv(gitlabPipelineNameEnvVar)
	if ciPipelineName != "" {
		attributes = append(attributes, attribute.String(string(semconv.CICDPipelineNameKey), ciPipelineName))
	}

	ciJobId := os.Getenv(gitlabJobIdEnvVar)
	if ciJobId != "" {
		attributes = append(attributes, attribute.String(string(semconv.CICDPipelineTaskRunIDKey), ciJobId))
	}

	ciJobName := os.Getenv(gitlabJobNameEnvVar)
	if ciJobName != "" {
		attributes = append(attributes, attribute.String(string(semconv.CICDPipelineTaskNameKey), ciJobName))
	}

	ciJobStage := os.Getenv(gitlabJobStageEnvVar)
	if ciJobStage != "" {
		attributes = append(attributes, attribute.String(string(semconv.CICDPipelineTaskTypeKey), ciJobStage))
	}

	ciPipelineId := os.Getenv(gitlabPipelineIdEnvVar)
	if ciPipelineId != "" {
		attributes = append(attributes, attribute.String(string(semconv.CICDPipelineRunIDKey), ciPipelineId))
	}

	ciPipelineUrl := os.Getenv(gitlabJobUrlEnvVar)
	if ciPipelineUrl != "" {
		attributes = append(attributes, attribute.String(string(semconv.CICDPipelineTaskRunURLFullKey), ciPipelineUrl))
	}
	return attributes
}

// detectVCSAttributes https://github.com/open-telemetry/semantic-conventions/blob/main/docs/attributes-registry/vcs.md
func detectVCSAttributes() []attribute.KeyValue {
	var attributes []attribute.KeyValue

	ciRefName := os.Getenv(gitlabCommitRefNameEnvVar)
	if ciRefName != "" {
		attributes = append(attributes, attribute.String(string(semconv.VCSRepositoryRefNameKey), ciRefName))
	}

	ciTag := os.Getenv(gitlabCommitTagEnvVar)
	if ciTag != "" {
		attributes = append(attributes, semconv.VCSRepositoryRefTypeTag)
	} else {
		attributes = append(attributes, semconv.VCSRepositoryRefTypeBranch)
	}

	mrID := os.Getenv(gitlabMergeRequestIIDEnvVar)
	if mrID != "" {
		attributes = append(attributes, attribute.String(string(semconv.VCSRepositoryChangeIDKey), mrID))
	}

	projectUrl := os.Getenv(gitlabProjectUrlEnvVar)
	if projectUrl != "" {
		attributes = append(attributes, attribute.String(string(semconv.VCSRepositoryURLFullKey), projectUrl))
	}

	// There is no SemConv for the ProjectID var
	//projectID := os.Getenv(gitlabProjectIDEnvVar)
	//if projectID != "" {
	//	attributes = append(attributes, attribute.String(string(semconv.VCSRepositoryProjectID), projectID))
	//}

	return attributes
}
