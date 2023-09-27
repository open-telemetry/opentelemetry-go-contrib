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
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// S3AttributeSetter sets S3 specific attributes depending on the S3 operation being performed.
func S3AttributeSetter(ctx context.Context, in middleware.InitializeInput) []attribute.KeyValue {
	s3Attributes := []attribute.KeyValue{}

	switch v := in.Parameters.(type) {
	case *s3.AbortMultipartUploadInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

		s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(aws.ToString(v.UploadId)))

	case *s3.CompleteMultipartUploadInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

		s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(aws.ToString(v.UploadId)))

	case *s3.CopyObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3CopySource(aws.ToString(v.CopySource)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.CreateBucketInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.CreateMultipartUploadInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.DeleteBucketInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketAnalyticsConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketCorsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketEncryptionInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketIntelligentTieringConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketInventoryConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketLifecycleInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketMetricsConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketOwnershipControlsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketPolicyInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketReplicationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketTaggingInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteBucketWebsiteInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.DeleteObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.DeleteObjectsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Delete(s3ShorthandSerializeDelete(v.Delete)))

	case *s3.DeleteObjectTaggingInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.DeletePublicAccessBlockInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketAccelerateConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketAclInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketAnalyticsConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketCorsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketEncryptionInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketIntelligentTieringConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketInventoryConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketLifecycleConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketLocationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketLoggingInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketMetricsConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketNotificationConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketOwnershipControlsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketPolicyInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketPolicyStatusInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketReplicationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketRequestPaymentInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketTaggingInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketVersioningInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetBucketWebsiteInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.GetObjectAclInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.GetObjectAttributesInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.GetObjectLegalHoldInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.GetObjectLockConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.GetObjectRetentionInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.GetObjectTaggingInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.GetObjectTorrentInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.GetPublicAccessBlockInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.HeadBucketInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.HeadObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.ListBucketAnalyticsConfigurationsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.ListBucketIntelligentTieringConfigurationsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.ListBucketInventoryConfigurationsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.ListBucketMetricsConfigurationsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.ListBucketsInput:
		// ListBucketsInput defines no attributes

	case *s3.ListMultipartUploadsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.ListObjectsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.ListObjectsV2Input:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.ListObjectVersionsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.ListPartsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

		s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(aws.ToString(v.UploadId)))

	case *s3.PutBucketAccelerateConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketAclInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketAnalyticsConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketCorsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketEncryptionInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketIntelligentTieringConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketInventoryConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketLifecycleConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketLoggingInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketMetricsConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketNotificationConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketOwnershipControlsInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketPolicyInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketReplicationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketRequestPaymentInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketTaggingInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketVersioningInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutBucketWebsiteInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.PutObjectAclInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.PutObjectLegalHoldInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.PutObjectLockConfigurationInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.PutObjectRetentionInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.PutObjectTaggingInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.PutPublicAccessBlockInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

	case *s3.RestoreObjectInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.SelectObjectContentInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

	case *s3.UploadPartInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

		s3Attributes = append(s3Attributes, semconv.AWSS3PartNumber(int(v.PartNumber)))

		s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(aws.ToString(v.UploadId)))

	case *s3.UploadPartCopyInput:
		s3Attributes = append(s3Attributes, semconv.AWSS3Bucket(aws.ToString(v.Bucket)))

		s3Attributes = append(s3Attributes, semconv.AWSS3Key(aws.ToString(v.Key)))

		s3Attributes = append(s3Attributes, semconv.AWSS3PartNumber(int(v.PartNumber)))

		s3Attributes = append(s3Attributes, semconv.AWSS3UploadID(aws.ToString(v.UploadId)))

	case *s3.WriteGetObjectResponseInput:
		// WriteGetObjectResponseInput defines no attributes
	}

	return s3Attributes
}

// s3ShorthandSerializeDelete serializes the Delete struct into shorthand syntax
// https://docs.aws.amazon.com/cli/latest/userguide/cli-usage-shorthand.html
func s3ShorthandSerializeDelete(d *s3types.Delete) string {
	var builder strings.Builder

	fmt.Fprint(&builder, "Objects=[")
	count := len(d.Objects)
	for i, object := range d.Objects {
		fmt.Fprint(&builder, "{")

		fmt.Fprintf(&builder, "Key=%s", aws.ToString(object.Key))

		if object.VersionId != nil {
			fmt.Fprintf(&builder, ",VersionId=%s", aws.ToString(object.VersionId))
		}

		fmt.Fprint(&builder, "}")
		if i+1 != count {
			fmt.Fprint(&builder, ",")
		}
	}
	fmt.Fprint(&builder, "],")

	fmt.Fprintf(&builder, "Quiet=%t", d.Quiet)

	return builder.String()
}
