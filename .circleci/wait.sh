#!/bin/bash

wait_for_cassandra () {
  for i in 1 2 3 4 5; do
    if docker exec $1 nodetool status | grep "^UN"; then
      exit 0
    fi
    echo "Cassandra not yet available"
    sleep 10
  done
  echo "Timeout waiting for cassandra to initialize"
  exit 1
}

wait_for_mongo () {
  for i in 1 2 3 4 5; do
    if docker exec $1 mongo; then
      exit 0
    fi
    echo "Mongo not yet available..."
    sleep 10
  done
  echo "Timeout waiting for mongo to initialize"
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
  wait_for_cassandra $IMG_NAME
elif [ "$CMD" == "mongo" ]; then
  wait_for_mongo $IMG_NAME
else
  echo "unknown CMD"
  exit 1
fi
