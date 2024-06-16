#!/bin/bash

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

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
elif [ "$CMD" == "gomemcache" ]; then
  wait_for_gomemcache
else
  echo "unknown CMD"
  exit 1
fi
