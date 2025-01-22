// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws // import "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/smithy-go/middleware"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// DynamoDBAttributeSetter sets DynamoDB specific attributes depending on the DynamoDB operation being performed.
//
// Deprecated: Use DynamoDBAttributeBuilder instead. This will be removed in a future release.
func DynamoDBAttributeSetter(ctx context.Context, in middleware.InitializeInput) []attribute.KeyValue {
	return DynamoDBAttributeBuilder(ctx, in, middleware.InitializeOutput{})
}

// DynamoDBAttributeBuilder sets DynamoDB specific attributes depending on the DynamoDB operation being performed.
func DynamoDBAttributeBuilder(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput) []attribute.KeyValue {
	dynamodbAttributes := []attribute.KeyValue{semconv.DBSystemDynamoDB}

	switch v := in.Parameters.(type) {
	case *dynamodb.GetItemInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

		if v.ConsistentRead != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBConsistentRead(*v.ConsistentRead))
		}

		if v.ProjectionExpression != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProjection(*v.ProjectionExpression))
		}

	case *dynamodb.BatchGetItemInput:
		var tableNames []string
		for k := range v.RequestItems {
			tableNames = append(tableNames, k)
		}
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(tableNames...))

	case *dynamodb.BatchWriteItemInput:
		var tableNames []string
		for k := range v.RequestItems {
			tableNames = append(tableNames, k)
		}
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(tableNames...))

	case *dynamodb.CreateTableInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

		if v.GlobalSecondaryIndexes != nil {
			var idx []string
			for _, gsi := range v.GlobalSecondaryIndexes {
				i, _ := json.Marshal(gsi)
				idx = append(idx, string(i))
			}
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBGlobalSecondaryIndexes(idx...))
		}

		if v.LocalSecondaryIndexes != nil {
			var idx []string
			for _, lsi := range v.LocalSecondaryIndexes {
				i, _ := json.Marshal(lsi)
				idx = append(idx, string(i))
			}
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBLocalSecondaryIndexes(idx...))
		}

		if v.ProvisionedThroughput != nil {
			read := float64(*v.ProvisionedThroughput.ReadCapacityUnits)
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProvisionedReadCapacity(read))
			write := float64(*v.ProvisionedThroughput.WriteCapacityUnits)
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProvisionedWriteCapacity(write))
		}

	case *dynamodb.DeleteItemInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

	case *dynamodb.DeleteTableInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

	case *dynamodb.DescribeTableInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

	case *dynamodb.ListTablesInput:
		if v.ExclusiveStartTableName != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBExclusiveStartTable(*v.ExclusiveStartTableName))
		}

		if v.Limit != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBLimit(int(*v.Limit)))
		}

	case *dynamodb.PutItemInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

	case *dynamodb.QueryInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

		if v.ConsistentRead != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBConsistentRead(*v.ConsistentRead))
		}

		if v.IndexName != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBIndexName(*v.IndexName))
		}

		if v.Limit != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBLimit(int(*v.Limit)))
		}

		if v.ScanIndexForward != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBScanForward(*v.ScanIndexForward))
		}

		if v.ProjectionExpression != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProjection(*v.ProjectionExpression))
		}

		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBSelect(string(v.Select)))

	case *dynamodb.ScanInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

		if v.ConsistentRead != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBConsistentRead(*v.ConsistentRead))
		}

		if v.IndexName != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBIndexName(*v.IndexName))
		}

		if v.Limit != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBLimit(int(*v.Limit)))
		}

		if v.ProjectionExpression != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProjection(*v.ProjectionExpression))
		}

		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBSelect(string(v.Select)))

		if v.Segment != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBSegment(int(*v.Segment)))
		}

		if v.TotalSegments != nil {
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTotalSegments(int(*v.TotalSegments)))
		}

	case *dynamodb.UpdateItemInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

	case *dynamodb.UpdateTableInput:
		dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBTableNames(*v.TableName))

		if v.AttributeDefinitions != nil {
			var def []string
			for _, ad := range v.AttributeDefinitions {
				d, _ := json.Marshal(ad)
				def = append(def, string(d))
			}
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBAttributeDefinitions(def...))
		}

		if v.GlobalSecondaryIndexUpdates != nil {
			var idx []string
			for _, gsiu := range v.GlobalSecondaryIndexUpdates {
				i, _ := json.Marshal(gsiu)
				idx = append(idx, string(i))
			}
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBGlobalSecondaryIndexUpdates(idx...))
		}

		if v.ProvisionedThroughput != nil {
			read := float64(*v.ProvisionedThroughput.ReadCapacityUnits)
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProvisionedReadCapacity(read))
			write := float64(*v.ProvisionedThroughput.WriteCapacityUnits)
			dynamodbAttributes = append(dynamodbAttributes, semconv.AWSDynamoDBProvisionedWriteCapacity(write))
		}
	}

	return dynamodbAttributes
}
