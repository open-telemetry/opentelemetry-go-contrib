// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
)

func TestS3AttributesAbortMultipartUploadInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.AbortMultipartUploadInput{
			Bucket:   aws.String("test-bucket"),
			Key:      aws.String("test-key"),
			UploadId: aws.String("abcd"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
			attribute.String("aws.s3.upload_id", "abcd"),
		},
	)
}

func TestS3AttributesCompleteMultipartUploadInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.CompleteMultipartUploadInput{
			Bucket:   aws.String("test-bucket"),
			Key:      aws.String("test-key"),
			UploadId: aws.String("abcd"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
			attribute.String("aws.s3.upload_id", "abcd"),
		},
	)
}

func TestS3AttributesCopyObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.CopyObjectInput{
			Bucket:     aws.String("test-bucket"),
			CopySource: aws.String("test-source"),
			Key:        aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.copy_source", "test-source"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesCreateBucketInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.CreateBucketInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesCreateMultipartUploadInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.CreateMultipartUploadInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesDeleteBucketInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketAnalyticsConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketAnalyticsConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketCorsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketCorsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketEncryptionInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketEncryptionInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketIntelligentTieringConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketIntelligentTieringConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketInventoryConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketInventoryConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketLifecycleInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketLifecycleInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketMetricsConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketMetricsConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketOwnershipControlsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketOwnershipControlsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketPolicyInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketPolicyInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketReplicationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketReplicationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketTaggingInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketTaggingInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteBucketWebsiteInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteBucketWebsiteInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesDeleteObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesDeleteObjectsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteObjectsInput{
			Bucket: aws.String("test-bucket"),
			Delete: &s3types.Delete{
				Objects: []s3types.ObjectIdentifier{
					{
						Key:       aws.String("test-key"),
						VersionId: nil,
					},
				},
				Quiet: false,
			},
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.delete", "Objects=[{Key=test-key}],Quiet=false"),
		},
	)
}

func TestS3AttributesDeleteObjectTaggingInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeleteObjectTaggingInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesDeletePublicAccessBlockInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.DeletePublicAccessBlockInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketAccelerateConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketAccelerateConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketAclInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketAclInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketAnalyticsConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketAnalyticsConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketCorsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketCorsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketEncryptionInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketEncryptionInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketIntelligentTieringConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketIntelligentTieringConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketInventoryConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketInventoryConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketLifecycleConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketLifecycleConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketLocationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketLocationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketLoggingInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketLoggingInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketMetricsConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketMetricsConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketNotificationConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketNotificationConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketOwnershipControlsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketOwnershipControlsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketPolicyInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketPolicyInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketPolicyStatusInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketPolicyStatusInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketReplicationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketReplicationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketRequestPaymentInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketRequestPaymentInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketTaggingInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketTaggingInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketVersioningInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketVersioningInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetBucketWebsiteInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetBucketWebsiteInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesGetObjectAclInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetObjectAclInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesGetObjectAttributesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetObjectAttributesInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesGetObjectLegalHoldInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetObjectLegalHoldInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesGetObjectLockConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetObjectLockConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesGetObjectRetentionInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetObjectRetentionInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesGetObjectTaggingInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetObjectTaggingInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesGetObjectTorrentInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetObjectTorrentInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesGetPublicAccessBlockInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.GetPublicAccessBlockInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesHeadBucketInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.HeadBucketInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesHeadObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.HeadObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesListBucketAnalyticsConfigurationsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListBucketAnalyticsConfigurationsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesListBucketIntelligentTieringConfigurationsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListBucketIntelligentTieringConfigurationsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesListBucketInventoryConfigurationsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListBucketInventoryConfigurationsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesListBucketMetricsConfigurationsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListBucketMetricsConfigurationsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesListBucketsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListBucketsInput{},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{},
	)
}

func TestS3AttributesListMultipartUploadsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListMultipartUploadsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesListObjectsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListObjectsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesListObjectsV2Input(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListObjectsV2Input{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesListObjectVersionsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListObjectVersionsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesListPartsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.ListPartsInput{
			Bucket:   aws.String("test-bucket"),
			Key:      aws.String("test-key"),
			UploadId: aws.String("abcd"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
			attribute.String("aws.s3.upload_id", "abcd"),
		},
	)
}

func TestS3AttributesPutBucketAccelerateConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketAccelerateConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketAclInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketAclInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketAnalyticsConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketAnalyticsConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketCorsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketCorsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketEncryptionInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketEncryptionInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketIntelligentTieringConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketIntelligentTieringConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketInventoryConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketInventoryConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketLifecycleConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketLifecycleConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketLoggingInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketLoggingInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketMetricsConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketMetricsConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketNotificationConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketNotificationConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketOwnershipControlsInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketOwnershipControlsInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketPolicyInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketPolicyInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketReplicationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketReplicationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketRequestPaymentInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketRequestPaymentInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketTaggingInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketTaggingInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketVersioningInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketVersioningInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutBucketWebsiteInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutBucketWebsiteInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesPutObjectAclInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutObjectAclInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesPutObjectLegalHoldInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutObjectLegalHoldInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesPutObjectLockConfigurationInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutObjectLockConfigurationInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesPutObjectRetentionInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutObjectRetentionInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesPutObjectTaggingInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutObjectTaggingInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesPutPublicAccessBlockInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.PutPublicAccessBlockInput{
			Bucket: aws.String("test-bucket"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
		},
	)
}

func TestS3AttributesRestoreObjectInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.RestoreObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesSelectObjectContentInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.SelectObjectContentInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test-key"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
		},
	)
}

func TestS3AttributesUploadPartInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.UploadPartInput{
			Bucket:     aws.String("test-bucket"),
			Key:        aws.String("test-key"),
			PartNumber: 1234,
			UploadId:   aws.String("abcd"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
			attribute.Int("aws.s3.part_number", 1234),
			attribute.String("aws.s3.upload_id", "abcd"),
		},
	)
}

func TestS3AttributesUploadPartCopyInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.UploadPartCopyInput{
			Bucket:     aws.String("test-bucket"),
			Key:        aws.String("test-key"),
			PartNumber: 1234,
			UploadId:   aws.String("abcd"),
		},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{
			attribute.String("aws.s3.bucket", "test-bucket"),
			attribute.String("aws.s3.key", "test-key"),
			attribute.Int("aws.s3.part_number", 1234),
			attribute.String("aws.s3.upload_id", "abcd"),
		},
	)
}

func TestS3AttributesWriteGetObjectResponseInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &s3.WriteGetObjectResponseInput{},
	}

	attributes := S3AttributeSetter(context.TODO(), input)

	assert.ElementsMatch(t, attributes,
		[]attribute.KeyValue{},
	)
}

func TestS3AttributesShorthandSerializeDelete(t *testing.T) {
	testcases := map[string]struct {
		input    *s3types.Delete
		expected string
	}{
		"single no version not quiet": {
			input: &s3types.Delete{
				Objects: []s3types.ObjectIdentifier{
					{
						Key:       aws.String("test-key"),
						VersionId: nil,
					},
				},
				Quiet: false,
			},
			expected: "Objects=[{Key=test-key}],Quiet=false",
		},
		"single version quiet": {
			input: &s3types.Delete{
				Objects: []s3types.ObjectIdentifier{
					{
						Key:       aws.String("test-key"),
						VersionId: aws.String("abc123"),
					},
				},
				Quiet: true,
			},
			expected: "Objects=[{Key=test-key,VersionId=abc123}],Quiet=true",
		},
		"multiple version quiet": {
			input: &s3types.Delete{
				Objects: []s3types.ObjectIdentifier{
					{
						Key:       aws.String("test-key1"),
						VersionId: aws.String("abc123"),
					},
					{
						Key:       aws.String("test-key2"),
						VersionId: aws.String("xyz789"),
					},
				},
				Quiet: true,
			},
			expected: "Objects=[{Key=test-key1,VersionId=abc123},{Key=test-key2,VersionId=xyz789}],Quiet=true",
		},
	}

	for name, testcase := range testcases {
		t.Run(name, func(t *testing.T) {
			out := s3ShorthandSerializeDelete(testcase.input)

			if a, e := out, testcase.expected; a != e {
				t.Fatalf("expected %q, got %q", e, a)
			}
		})
	}
}
