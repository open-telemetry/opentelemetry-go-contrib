#!/bin/sh

# Copyright The OpenTelemetry Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# constants
LAMBDA_FUNCTION_NAME=SampleLambdaGo
ROLE_NAME="$LAMBDA_FUNCTION_NAME"Role
POLICY_NAME="$LAMBDA_FUNCTION_NAME"Policy
LOG_GROUP_NAME=/aws/lambda/"$LAMBDA_FUNCTION_NAME"
AWS_ACCT_ID=$(aws sts get-caller-identity | jq '.Account | tonumber')
ERROR_LOG_FILE=manualAWSCleanupErrors.log

# Clear log
rm $ERROR_LOG_FILE 2> /dev/null

# clear log group of all streams
if aws logs describe-log-streams --log-group-name "$LOG_GROUP_NAME" > /dev/null 2>> $ERROR_LOG_FILE ; then
  LOG_STREAM_NAME=$(aws logs describe-log-streams --log-group-name "$LOG_GROUP_NAME" --order-by LastEventTime --descending | jq --raw-output '.logStreams[0].logStreamName')
  while [ "$LOG_STREAM_NAME" != "null" ] ; do
    aws logs delete-log-stream --log-group-name "$LOG_GROUP_NAME" --log-stream-name "$LOG_STREAM_NAME" 2>> $ERROR_LOG_FILE && echo "Deleted log stream $LOG_STREAM_NAME"
    LOG_STREAM_NAME=$(aws logs describe-log-streams --log-group-name "$LOG_GROUP_NAME" --order-by LastEventTime --descending | jq --raw-output '.logStreams[0].logStreamName')
  done
  aws logs delete-log-group --log-group-name "$LOG_GROUP_NAME" && echo "Deleted log group $LOG_GROUP_NAME"
else
  echo "Did not delete log group, likely already deleted"
fi

# destroy remaining lambda resources if they exist
aws lambda delete-function --function-name "$LAMBDA_FUNCTION_NAME" 2>> $ERROR_LOG_FILE && echo "Deleted Lambda Function $LAMBDA_FUNCTION_NAME" || echo "Did not delete function, likely already deleted"
aws iam detach-role-policy --role-name "$ROLE_NAME" --policy-arn arn:aws:iam::"$AWS_ACCT_ID":policy/"$POLICY_NAME" 2>> $ERROR_LOG_FILE && echo "Detached $POLICY_NAME from $ROLE_NAME" || echo "Did not detach policy from role, likely already detached"
aws iam delete-policy --policy-arn arn:aws:iam::"$AWS_ACCT_ID":policy/"$POLICY_NAME" 2>> $ERROR_LOG_FILE && echo "Deleted IAM Policy POLICY_NAME" || echo "Did not delete IAM Policy, likely already deleted"
aws iam delete-role --role-name "$ROLE_NAME" 2>> $ERROR_LOG_FILE && echo "Deleted IAM Role $ROLE_NAME" || echo "Did not delete IAM Role, likely already deleted"

if [ -s $ERROR_LOG_FILE ] ; then
  echo 'Some resources failed to delete. Can ensure these errors were due to the resources existing by checking "'$ERROR_LOG_FILE'"'
fi