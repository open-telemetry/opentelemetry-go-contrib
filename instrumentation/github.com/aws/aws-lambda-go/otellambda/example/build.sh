#!/bin/sh

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# constants
LAMBDA_FUNCTION_NAME=SampleLambdaGo
ROLE_NAME="$LAMBDA_FUNCTION_NAME"Role
POLICY_NAME="$LAMBDA_FUNCTION_NAME"Policy
LOG_GROUP_NAME=/aws/lambda/"$LAMBDA_FUNCTION_NAME"
AWS_ACCT_ID=$(aws sts get-caller-identity | jq '.Account | tonumber')
MAX_CREATE_TRIES=5
MAX_GET_LOG_STREAM_TRIES=10

# build go executable
echo "1/6 Building go executable"
GOOS=linux GOARCH=amd64 go build -o ./build/bootstrap . > /dev/null
cd build || exit
zip bootstrap.zip bootstrap > /dev/null

# create AWS resources
echo "2/6 Creating necessary resources in AWS"
aws iam create-role --role-name "$ROLE_NAME" --assume-role-policy-document file://../assumeRolePolicyDocument.json > /dev/null
aws iam create-policy --policy-name "$POLICY_NAME" --policy-document file://../policyForRoleDocument.json > /dev/null
aws iam attach-role-policy --role-name "$ROLE_NAME" --policy-arn arn:aws:iam::"$AWS_ACCT_ID":policy/"$POLICY_NAME" > /dev/null
aws logs create-log-group --log-group-name "$LOG_GROUP_NAME" > /dev/null

# race condition exists such that a role can be created and validated
# via IAM, yet still cannot be assumed by Lambda, we will retry up to
# MAX_CREATE_TRIES times to create the function
TIMEOUT="$MAX_CREATE_TRIES"
CREATE_FUNCTION_SUCCESS=$(aws lambda create-function --function-name "$LAMBDA_FUNCTION_NAME" --runtime provided.al2 --handler bootstrap --zip-file fileb://bootstrap.zip --role arn:aws:iam::"$AWS_ACCT_ID":role/"$ROLE_NAME" --timeout 5 --tracing-config Mode=Active > /dev/null || echo "false")
while [ "$CREATE_FUNCTION_SUCCESS" = "false" ] && [ "$TIMEOUT" -ne 1 ] ; do
  echo "    Retrying create-function, role likely not ready for use..."
  sleep 1
  TIMEOUT=$((TIMEOUT - 1))
  CREATE_FUNCTION_SUCCESS=$(aws lambda create-function --function-name "$LAMBDA_FUNCTION_NAME" --runtime provided.al2 --handler bootstrap --zip-file fileb://bootstrap.zip --role arn:aws:iam::"$AWS_ACCT_ID":role/"$ROLE_NAME" --timeout 5 --tracing-config Mode=Active > /dev/null || echo "false")
done
if [ "$TIMEOUT" -eq 1 ] ; then
  echo "Error: max retries reached when attempting to create Lambda Function"
fi

# invoke lambda
echo "3/6 Invoking lambda"
aws lambda invoke --function-name "$LAMBDA_FUNCTION_NAME" --payload "" resp.json

# get logs from lambda (via cloudwatch)
# logs sent from lambda to Cloudwatch and retrieved
# from there because example logs are too long to
# return directly from lambda invocation
echo "4/6 Storing logs from AWS"

# significant (3+ second) delay can occur between invoking Lambda and
# the related log stream existing in Cloudwatch. We will retry to
# retrieve the log stream up to MAX_GET_LOG_STREAM_TRIES
TIMEOUT="$MAX_GET_LOG_STREAM_TRIES"
LOG_STREAM_NAME=$(aws logs describe-log-streams --log-group-name "$LOG_GROUP_NAME" --order-by LastEventTime --descending | jq --raw-output '.logStreams[0].logStreamName')
while [ "$LOG_STREAM_NAME" = "null" ] && [ "$TIMEOUT" -ne 1 ] ; do
  echo "    Waiting for log stream to be created..."
  sleep 1
  TIMEOUT=$((TIMEOUT - 1))
  LOG_STREAM_NAME=$(aws logs describe-log-streams --log-group-name "$LOG_GROUP_NAME" --order-by LastEventTime --descending | jq --raw-output '.logStreams[0].logStreamName')
done
if [ "$TIMEOUT" -eq 1 ] ; then
  echo "Timed out waiting for log stream to be created"
fi

# minor (<1 second) delay can exist when adding logs to the
# log stream such that only partial logs will be returned.
# Will wait small amount of time to let logs fully populate
sleep 2
aws logs get-log-events --log-group-name "$LOG_GROUP_NAME" --log-stream-name "$LOG_STREAM_NAME" | jq --join-output '.events[] | select(has("message")) | .message' | jq -R -r '. as $line | try fromjson catch $line' > lambdaLogs

# destroy lambda resources
echo "5/6 Destroying AWS resources"
aws logs delete-log-stream --log-group-name "$LOG_GROUP_NAME" --log-stream-name "$LOG_STREAM_NAME"
aws logs delete-log-group --log-group-name "$LOG_GROUP_NAME"
aws lambda delete-function --function-name $LAMBDA_FUNCTION_NAME
aws iam detach-role-policy --role-name "$ROLE_NAME" --policy-arn arn:aws:iam::"$AWS_ACCT_ID":policy/"$POLICY_NAME"
aws iam delete-policy --policy-arn arn:aws:iam::"$AWS_ACCT_ID":policy/"$POLICY_NAME"
aws iam delete-role --role-name "$ROLE_NAME"

# display logs
printf "6/6 Displaying logs from AWS:\n\n\n"
cat lambdaLogs
