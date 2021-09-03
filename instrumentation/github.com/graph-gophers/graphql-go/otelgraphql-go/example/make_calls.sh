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

#!/bin/bash
ENDPOINT='http://graphql-go-server:8080/graphql'

wget --body-data '{"query":"query AllUserFullNames {users {fullName}}","variables":{}}' \
  --method GET \
  $ENDPOINT
wget --body-data '{"query":"query SingleUserInfo($username: String!) {user(username: $username) {organization\nfullName\nusername\n}}","variables":{"username":"johnsmith"}}' \
  --method GET \
  $ENDPOINT
wget --body-data '{"query":"query UsersInOrganization($organization: String!){usersOfOrganization(organization: $organization) {username}}","variables":{"organization":"HR"}}' \
  --method GET \
  $ENDPOINT
wget --body-data '{"query":"mutation NewUser($user: UserInput!) {createUser(userInput: $user){username}}","variables":{"user":{"username":"joesmith","fullName":"Joe Smith","organization":"Marketing"}}}' \
  --method GET \
  $ENDPOINT
