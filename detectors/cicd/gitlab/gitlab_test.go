package gitlab

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"testing"
)

type envPair struct {
	Key   string
	Value string
}

func setTestEnv(t *testing.T, envs []envPair) {
	for _, env := range envs {
		t.Setenv(env.Key, env.Value)
	}
}

func TestGitlabDetector(t *testing.T) {

	tcs := []struct {
		scenario         string
		envs             []envPair
		expectedError    error
		expectedResource *resource.Resource
	}{
		{
			scenario: "all env configured",
			envs: []envPair{
				{"CI", "true"},
				{"GITLAB_CI", "true"},
				{"CI_PIPELINE_NAME", "pipeline_name"},
				{"CI_JOB_ID", "123"},
				{"CI_JOB_NAME", "test something"},
				{"CI_JOB_STAGE", "test"},
				{"CI_PIPELINE_ID", "12345"},
				{"CI_JOB_URL", "https://gitlab/job/123"},
				{"CI_COMMIT_REF_NAME", "abc123"},
				{"CI_MERGE_REQUEST_IID", "12"},
				{"CI_PROJECT_URL", "https://gitlab/org/project"},
				{"CI_PROJECT_ID", "111"},
			},
			expectedError: nil,
			expectedResource: resource.NewWithAttributes(semconv.SchemaURL, []attribute.KeyValue{
				attribute.String(string(semconv.CICDPipelineNameKey), "pipeline_name"),
				attribute.String(string(semconv.CICDPipelineTaskRunIDKey), "123"),
				attribute.String(string(semconv.CICDPipelineTaskNameKey), "test something"),
				attribute.String(string(semconv.CICDPipelineTaskTypeKey), "test"),
				attribute.String(string(semconv.CICDPipelineRunIDKey), "12345"),
				attribute.String(string(semconv.CICDPipelineTaskRunURLFullKey), "https://gitlab/job/123"),
				attribute.String(string(semconv.VCSRepositoryRefNameKey), "abc123"),
				attribute.String(string(semconv.VCSRepositoryRefTypeKey), "branch"),
				attribute.String(string(semconv.VCSRepositoryChangeIDKey), "12"),
				attribute.String(string(semconv.VCSRepositoryURLFullKey), "https://gitlab/org/project"),
				//	attribute.String(string(semconv.VCSRepositoryProjectID), "111"),
			}...),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.scenario, func(t *testing.T) {
			setTestEnv(t, tc.envs)

			detector := NewResourceDetector()

			res, err := detector.Detect(context.Background())

			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedResource, res)
		})
	}

}
