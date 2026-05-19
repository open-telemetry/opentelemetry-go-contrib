#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
#

set -euo pipefail

if [[ -z "${COMPONENT:-}" || -z "${ISSUE:-}" ]]; then
    echo "Either COMPONENT or ISSUE has not been set, please ensure both are set."
    exit 0
fi

CUR_DIRECTORY=$(dirname "$0")

# Labels are formatted as "<type>: <component>" (e.g. "instrumentation: otelgrpc").
# Extract the component name (the part after ': ') and resolve the full path
# from CODEOWNERS, since get-codeowners.sh matches by path prefix.
COMPONENT_NAME=$(echo "${COMPONENT}" | sed 's/^[^:]*:[[:space:]]*//')
COMPONENT_PATH=$(grep -F "/${COMPONENT_NAME}/" "${CUR_DIRECTORY}/../CODEOWNERS" | awk '{ print $1 }' | sed 's|/$||' | head -1 || true)

if [[ -z "${COMPONENT_PATH}" ]]; then
    exit 0
fi

OWNERS=$(COMPONENT="${COMPONENT_PATH}" bash "${CUR_DIRECTORY}/get-codeowners.sh")

if [[ -z "${OWNERS}" ]]; then
    exit 0
fi

gh issue comment "${ISSUE}" --body "Pinging code owners for ${COMPONENT}: ${OWNERS}."
