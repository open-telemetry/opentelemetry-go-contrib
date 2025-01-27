// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelaws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
)

func TestDynamodbTagsBatchGetItemInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.BatchGetItemInput{
			RequestItems: map[string]dtypes.KeysAndAttributes{
				"table1": {
					Keys: []map[string]dtypes.AttributeValue{
						{
							"id": &dtypes.AttributeValueMemberS{Value: "123"},
						},
					},
				},
			},
		},
	}

	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice("aws.dynamodb.table_names", []string{"table1"}))
}

func TestDynamodbTagsBatchWriteItemInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]dtypes.WriteRequest{
				"table1": {
					{
						DeleteRequest: &dtypes.DeleteRequest{
							Key: map[string]dtypes.AttributeValue{
								"id": &dtypes.AttributeValueMemberS{Value: "123"},
							},
						},
					},
					{
						PutRequest: &dtypes.PutRequest{
							Item: map[string]dtypes.AttributeValue{
								"id": &dtypes.AttributeValueMemberS{Value: "234"},
							},
						},
					},
				},
			},
		},
	}

	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice("aws.dynamodb.table_names", []string{"table1"}))
}

func TestDynamodbTagsCreateTableInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.CreateTableInput{
			AttributeDefinitions: []dtypes.AttributeDefinition{
				{
					AttributeName: aws.String("id"),
					AttributeType: dtypes.ScalarAttributeTypeS,
				},
			},
			KeySchema: []dtypes.KeySchemaElement{
				{
					AttributeName: aws.String("id"),
					KeyType:       dtypes.KeyTypeHash,
				},
			},
			TableName:   aws.String("table1"),
			BillingMode: dtypes.BillingModePayPerRequest,
			ProvisionedThroughput: &dtypes.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(123),
				WriteCapacityUnits: aws.Int64(456),
			},
			GlobalSecondaryIndexes: []dtypes.GlobalSecondaryIndex{
				{
					IndexName: aws.String("index1"),
					KeySchema: []dtypes.KeySchemaElement{
						{
							AttributeName: aws.String("attributename"),
							KeyType:       dtypes.KeyTypeHash,
						},
					},
					Projection: &dtypes.Projection{
						NonKeyAttributes: []string{"non-key-attributes"},
					},
				},
			},
			LocalSecondaryIndexes: []dtypes.LocalSecondaryIndex{
				{
					IndexName: aws.String("index2"),
					KeySchema: []dtypes.KeySchemaElement{
						{
							AttributeName: aws.String("attributename"),
							KeyType:       dtypes.KeyTypeHash,
						},
					},
				},
			},
		},
	}

	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.table_names", []string{"table1"},
	))
	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.global_secondary_indexes",
		[]string{
			`{"IndexName":"index1","KeySchema":[{"AttributeName":"attributename","KeyType":"HASH"}],"Projection":{"NonKeyAttributes":["non-key-attributes"],"ProjectionType":""},"OnDemandThroughput":null,"ProvisionedThroughput":null,"WarmThroughput":null}`,
		},
	))
	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.local_secondary_indexes",
		[]string{
			`{"IndexName":"index2","KeySchema":[{"AttributeName":"attributename","KeyType":"HASH"}],"Projection":null}`,
		},
	))
	assert.Contains(t, attributes, attribute.Float64("aws.dynamodb.provisioned_read_capacity", 123))
	assert.Contains(t, attributes, attribute.Float64("aws.dynamodb.provisioned_write_capacity", 456))
}

func TestDynamodbTagsDeleteItemInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.DeleteItemInput{
			Key: map[string]dtypes.AttributeValue{
				"id": &dtypes.AttributeValueMemberS{Value: "123"},
			},
			TableName: aws.String("table1"),
		},
	}
	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.table_names", []string{"table1"},
	))
}

func TestDynamodbTagsDeleteTableInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.DeleteTableInput{
			TableName: aws.String("table1"),
		},
	}
	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.table_names", []string{"table1"},
	))
}

func TestDynamodbTagsDescribeTableInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.DescribeTableInput{
			TableName: aws.String("table1"),
		},
	}
	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.table_names", []string{"table1"},
	))
}

func TestDynamodbTagsListTablesInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.ListTablesInput{
			ExclusiveStartTableName: aws.String("table1"),
			Limit:                   aws.Int32(10),
		},
	}
	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.String("aws.dynamodb.exclusive_start_table", "table1"))
	assert.Contains(t, attributes, attribute.Int("aws.dynamodb.limit", 10))
}

func TestDynamodbTagsPutItemInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.PutItemInput{
			TableName: aws.String("table1"),
			Item: map[string]dtypes.AttributeValue{
				"id":    &dtypes.AttributeValueMemberS{Value: "12346"},
				"name":  &dtypes.AttributeValueMemberS{Value: "John Doe"},
				"email": &dtypes.AttributeValueMemberS{Value: "john@doe.io"},
			},
		},
	}

	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.table_names", []string{"table1"},
	))
}

func TestDynamodbTagsQueryInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.QueryInput{
			TableName:              aws.String("table1"),
			IndexName:              aws.String("index1"),
			ConsistentRead:         aws.Bool(true),
			Limit:                  aws.Int32(10),
			ScanIndexForward:       aws.Bool(true),
			ProjectionExpression:   aws.String("projectionexpression"),
			Select:                 dtypes.SelectAllAttributes,
			KeyConditionExpression: aws.String("id = :hashKey and #date > :rangeKey"),
			ExpressionAttributeNames: map[string]string{
				"#date": "date",
			},
			ExpressionAttributeValues: map[string]dtypes.AttributeValue{
				":hashKey":  &dtypes.AttributeValueMemberS{Value: "123"},
				":rangeKey": &dtypes.AttributeValueMemberN{Value: "20150101"},
			},
		},
	}

	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.table_names", []string{"table1"},
	))
	assert.Contains(t, attributes, attribute.Bool("aws.dynamodb.consistent_read", true))
	assert.Contains(t, attributes, attribute.String("aws.dynamodb.index_name", "index1"))
	assert.Contains(t, attributes, attribute.Int("aws.dynamodb.limit", 10))
	assert.Contains(t, attributes, attribute.Bool("aws.dynamodb.scan_forward", true))
	assert.Contains(t, attributes, attribute.String("aws.dynamodb.projection", "projectionexpression"))
	assert.Contains(t, attributes, attribute.String("aws.dynamodb.select", "ALL_ATTRIBUTES"))
}

func TestDynamodbTagsScanInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.ScanInput{
			TableName:            aws.String("my-table"),
			ConsistentRead:       aws.Bool(true),
			IndexName:            aws.String("index1"),
			Limit:                aws.Int32(10),
			ProjectionExpression: aws.String("Artist, Genre"),
			Segment:              aws.Int32(10),
			TotalSegments:        aws.Int32(100),
			Select:               dtypes.SelectAllAttributes,
		},
	}

	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.table_names", []string{"my-table"},
	))
	assert.Contains(t, attributes, attribute.Bool("aws.dynamodb.consistent_read", true))
	assert.Contains(t, attributes, attribute.String("aws.dynamodb.index_name", "index1"))
	assert.Contains(t, attributes, attribute.Int("aws.dynamodb.limit", 10))
	assert.Contains(t, attributes, attribute.String("aws.dynamodb.select", "ALL_ATTRIBUTES"))
	assert.Contains(t, attributes, attribute.Int("aws.dynamodb.total_segments", 100))
	assert.Contains(t, attributes, attribute.Int("aws.dynamodb.segment", 10))
	assert.Contains(t, attributes, attribute.String("aws.dynamodb.projection", "Artist, Genre"))
}

func TestDynamodbTagsUpdateItemInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.UpdateItemInput{
			TableName: aws.String("my-table"),
			Key: map[string]dtypes.AttributeValue{
				"id": &dtypes.AttributeValueMemberS{Value: "123"},
			},
			UpdateExpression: aws.String("set firstName = :firstName"),
			ExpressionAttributeValues: map[string]dtypes.AttributeValue{
				":firstName": &dtypes.AttributeValueMemberS{Value: "John McNewname"},
			},
		},
	}

	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.table_names", []string{"my-table"},
	))
}

func TestDynamodbTagsUpdateTableInput(t *testing.T) {
	input := middleware.InitializeInput{
		Parameters: &dynamodb.UpdateTableInput{
			TableName: aws.String("my-table"),
			AttributeDefinitions: []dtypes.AttributeDefinition{
				{
					AttributeName: aws.String("id"),
					AttributeType: dtypes.ScalarAttributeTypeS,
				},
			},
			GlobalSecondaryIndexUpdates: []dtypes.GlobalSecondaryIndexUpdate{
				{
					Create: &dtypes.CreateGlobalSecondaryIndexAction{
						IndexName: aws.String("index1"),
						KeySchema: []dtypes.KeySchemaElement{
							{
								AttributeName: aws.String("attribute"),
								KeyType:       dtypes.KeyTypeHash,
							},
						},
						Projection: &dtypes.Projection{
							NonKeyAttributes: []string{"attribute1", "attribute2"},
							ProjectionType:   dtypes.ProjectionTypeAll,
						},
					},
				},
			},
			ProvisionedThroughput: &dtypes.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(123),
				WriteCapacityUnits: aws.Int64(456),
			},
		},
	}

	attributes := DynamoDBAttributeBuilder(context.TODO(), input, middleware.InitializeOutput{})

	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.table_names", []string{"my-table"},
	))
	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.attribute_definitions",
		[]string{`{"AttributeName":"id","AttributeType":"S"}`},
	))
	assert.Contains(t, attributes, attribute.StringSlice(
		"aws.dynamodb.global_secondary_index_updates",
		[]string{
			`{"Create":{"IndexName":"index1","KeySchema":[{"AttributeName":"attribute","KeyType":"HASH"}],"Projection":{"NonKeyAttributes":["attribute1","attribute2"],"ProjectionType":"ALL"},"OnDemandThroughput":null,"ProvisionedThroughput":null,"WarmThroughput":null},"Delete":null,"Update":null}`,
		},
	))
	assert.Contains(t, attributes, attribute.Float64("aws.dynamodb.provisioned_read_capacity", 123))
	assert.Contains(t, attributes, attribute.Float64("aws.dynamodb.provisioned_write_capacity", 456))
}
