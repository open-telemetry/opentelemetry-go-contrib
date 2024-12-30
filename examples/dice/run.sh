# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

#!/bin/bash

# Check if at least one argument is provided
if [ -z "$1" ]; then
  echo "Usage: $0 {instrumented|uninstrumented}"
  exit 1
fi

# Switch based on the first argument
case "$1" in
  instrumented)
    echo "Running instrumented example..."
    cd instrumented || exit
    source tidy.sh
    source run.sh
    ;;
  uninstrumented)
    echo "Running uninstrumented example..."
    cd uninstrumented || exit
    source run.sh
    ;;
  *)
    echo "Invalid argument: $1. Use 'instrumented' or 'uninstrumented'."
    exit 1
    ;;
esac
