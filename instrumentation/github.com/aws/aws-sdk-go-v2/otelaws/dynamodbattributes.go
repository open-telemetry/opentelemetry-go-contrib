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
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// DynamoDBAttributeSetter sets DynamoDB specific attributes depending on the DynamoDB operation being performed.
func DynamoDBAttributeSetter(ctx context.Context, in middleware.InitializeInput) []attribute.KeyValue {
	dynamodbAttributes := []attribute.KeyValue{semconv.DBSystemKey.String("dynamodb")}

	switch v := in.Parameters.(type) {
	case *dynamodb.GetItemInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

		if v.ConsistentRead != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBConsistentReadKey.Bool(*v.ConsistentRead))
		}

		if v.ProjectionExpression != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProjectionKey.String(*v.ProjectionExpression))
		}

	case *dynamodb.BatchGetItemInput:
		var tableNames []string
		for k := range v.RequestItems {
			tableNames = append(tableNames, k)
		}
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.StringSlice(tableNames))

	case *dynamodb.BatchWriteItemInput:
		var tableNames []string
		for k := range v.RequestItems {
			tableNames = append(tableNames, k)
		}
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.StringSlice(tableNames))

	case *dynamodb.CreateTableInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

		if v.GlobalSecondaryIndexes != nil {
			globalindexes, _ := json.Marshal(v.GlobalSecondaryIndexes)
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBGlobalSecondaryIndexesKey.String(string(globalindexes)))
		}

		if v.LocalSecondaryIndexes != nil {
			localindexes, _ := json.Marshal(v.LocalSecondaryIndexes)
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBLocalSecondaryIndexesKey.String(string(localindexes)))
		}

		if v.ProvisionedThroughput != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProvisionedReadCapacityKey.Int64(*v.ProvisionedThroughput.ReadCapacityUnits))
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProvisionedWriteCapacityKey.Int64(*v.ProvisionedThroughput.WriteCapacityUnits))
		}

	case *dynamodb.DeleteItemInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

	case *dynamodb.DeleteTableInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

	case *dynamodb.DescribeTableInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

	case *dynamodb.ListTablesInput:
		if v.ExclusiveStartTableName != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBExclusiveStartTableKey.String(*v.ExclusiveStartTableName))
		}

		if v.Limit != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBLimitKey.Int(int(*v.Limit)))
		}

	case *dynamodb.PutItemInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

	case *dynamodb.QueryInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

		if v.ConsistentRead != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBConsistentReadKey.Bool(*v.ConsistentRead))
		}

		if v.IndexName != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBIndexNameKey.String(*v.IndexName))
		}

		if v.Limit != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBLimitKey.Int(int(*v.Limit)))
		}

		if v.ScanIndexForward != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBScanForwardKey.Bool(*v.ScanIndexForward))
		}

		if v.ProjectionExpression != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProjectionKey.String(*v.ProjectionExpression))
		}

		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBSelectKey.String(string(v.Select)))

	case *dynamodb.ScanInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

		if v.ConsistentRead != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBConsistentReadKey.Bool(*v.ConsistentRead))
		}

		if v.IndexName != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBIndexNameKey.String(*v.IndexName))
		}

		if v.Limit != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBLimitKey.Int(int(*v.Limit)))
		}

		if v.ProjectionExpression != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProjectionKey.String(*v.ProjectionExpression))
		}

		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBSelectKey.String(string(v.Select)))

		if v.Segment != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBSegmentKey.Int(int(*v.Segment)))
		}

		if v.TotalSegments != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTotalSegmentsKey.Int(int(*v.TotalSegments)))
		}

	case *dynamodb.UpdateItemInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

	case *dynamodb.UpdateTableInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNamesKey.String(*v.TableName))

		if v.AttributeDefinitions != nil {
			attributedefinitions, _ := json.Marshal(v.AttributeDefinitions)
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBAttributeDefinitionsKey.String(string(attributedefinitions)))
		}

		if v.GlobalSecondaryIndexUpdates != nil {
			globalsecondaryindexupdates, _ := json.Marshal(v.GlobalSecondaryIndexUpdates)
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBGlobalSecondaryIndexUpdatesKey.String(string(globalsecondaryindexupdates)))
		}

		if v.ProvisionedThroughput != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProvisionedReadCapacityKey.Int64(*v.ProvisionedThroughput.ReadCapacityUnits))
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProvisionedWriteCapacityKey.Int64(*v.ProvisionedThroughput.WriteCapacityUnits))
		}
	}

	return dynamodbAttributes
}
