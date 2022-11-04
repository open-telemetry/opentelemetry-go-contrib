#!/bin/bash

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

wait_for_cassandra () {
  for ((i = 0; i < 5; ++i)); do
    if docker exec "$1" nodetool status | grep "^UN"; then
      exit 0
    fi
    echo "Cassandra not yet available"
    sleep 10
  done
  echo "Timeout waiting for cassandra to initialize"
  exit 1
}

wait_for_mongo () {
  for ((i = 0; i < 5; ++i)); do
    if docker exec "$1" mongosh; then
      exit 0
    fi
    echo "Mongo not yet available..."
    sleep 10
  done
  echo "Timeout waiting for mongo to initialize"
  exit 1
}

wait_for_gomemcache () {
  for ((i = 0; i < 5; ++i)); do
    if nc -z localhost 11211; then
      exit 0
    fi
    echo "Gomemcache not yet available..."
    sleep 10
  done
  echo "Timeout waiting for gomemcache to initialize"
  exit 1
}

if [ -z "$CMD" ]; then
  echo "CMD is undefined. exiting..."
  exit 1
elif [ -z "$IMG_NAME" ]; then
  echo "IMG_NAME is undefined. exiting..."
  exit 1
fi

if [ "$CMD" == "cassandra" ]; then
  wait_for_cassandra "$IMG_NAME"
elif [ "$CMD" == "mongo" ]; then
  wait_for_mongo "$IMG_NAME"
elif [ "$CMD" == "gomemcache" ]; then
  wait_for_gomemcache
else
  echo "unknown CMD"
  exit 1
fi
