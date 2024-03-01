#!/bin/bash

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Returns 0 (true) when the current diff contains files in the provided
# target directory. TARGET should be a unique package name in the directory
# structure. For example, for the gocql integration, set TARGET=gocql so that
# a diff in any of the files in the instrumentation/gocql/gocql directory
# will be picked up by the grep. Diffs are compared against the main branch.

TARGET=$1

if [ -z "$TARGET" ]; then
  echo "TARGET is undefined"
  exit 1
fi

if git diff --name-only origin/main HEAD | grep -q "$TARGET"; then
  exit 0
else
  echo "no changes found for $TARGET. skipping tests..."
  exit 1
fi



