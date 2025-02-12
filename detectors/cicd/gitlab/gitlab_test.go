package gitlab

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"os"
	"testing"
)

type EnvPair struct {
	Key   string
	Value string
}

func setTestEnv(t *testing.T, envs []EnvPair) {
	for _, env := range envs {
		err := os.Setenv(env.Key, env.Value)
		if err != nil {
			t.Fatalf("Failed to set environment variable %s: %v", env.Key, err)
		}
	}
}

func TestGitlabDetector(t *testing.T) {

	tcs := []struct {
		scenario         string
		envs             []EnvPair
		expectedError    error
		expectedResource *resource.Resource
	}{
		{
			scenario: "all env configured",
			envs: []EnvPair{
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
			os.Clearenv()
			setTestEnv(t, tc.envs)

			detector := NewResourceDetector()

			res, err := detector.Detect(context.Background())

			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedResource, res)
		})
	}

}
