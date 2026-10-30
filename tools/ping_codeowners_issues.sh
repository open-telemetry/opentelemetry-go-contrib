#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
#

set -euo pipefail

CUR_DIRECTORY=$(dirname "$0")

# Labels are formatted as "<type>: <component>" (e.g. "instrumentation: otelgrpc").
resolve_component_path() {
    local component_name component_type
    component_type="${1%%:*}"
    component_name=$(printf '%s' "${1#*:}" | tr '[:upper:]' '[:lower:]' | sed 's/^[[:space:]]*//; s/[[:space:]]\+/:/g; s/x-ray/xray/g')

    case "${component_type}:${component_name}" in
        "detector:aws:ec2") echo "detectors/aws/ec2/v2" ;;
        "detector:aws:ecs") echo "detectors/aws/ecs" ;;
        "detector:aws:eks") echo "detectors/aws/eks" ;;
        "detector:aws:lambda") echo "detectors/aws/lambda" ;;
        "detector:gcp") echo "detectors/gcp" ;;
        "instrumentation:host") echo "instrumentation/host" ;;
        "instrumentation:otelaws") echo "instrumentation/github.com/aws/aws-sdk-go-v2/otelaws" ;;
        "instrumentation:otelgin") echo "instrumentation/github.com/gin-gonic/gin/otelgin" ;;
        "instrumentation:otelgrpc") echo "instrumentation/google.golang.org/grpc/otelgrpc" ;;
        "instrumentation:otelhttp") echo "instrumentation/net/http/otelhttp" ;;
        "instrumentation:otelhttptrace") echo "instrumentation/net/http/httptrace/otelhttptrace" ;;
        "instrumentation:otellambda") echo "instrumentation/github.com/aws/aws-lambda-go/otellambda" ;;
        "instrumentation:otelmongo") echo "instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo" ;;
        "instrumentation:otelmux") echo "instrumentation/github.com/gorilla/mux/otelmux" ;;
        "instrumentation:otelrestful") echo "instrumentation/github.com/emicklei/go-restful/otelrestful" ;;
        "instrumentation:runtime") echo "instrumentation/runtime" ;;
        "propagator:autoprop") echo "propagators/autoprop" ;;
        "propagator:aws"|"propagator:aws:xray") echo "propagators/aws" ;;
        "propagator:b3") echo "propagators/b3" ;;
        "propagator:jaeger") echo "propagators/jaeger" ;;
        "propagator:opencensus") echo "propagators/opencensus" ;;
        "propagator:ot") echo "propagators/ot" ;;
        "sampler:jaegerremote") echo "samplers/jaegerremote" ;;
        "sampler:probability:consistent") echo "samplers/probability/consistent" ;;
        "exporter:exporters/autoexport") echo "exporters/autoexport" ;;
        "zpages:zpages") echo "zpages" ;;
    esac
}

if [[ "${1:-}" == "--resolve" ]]; then
    resolve_component_path "${2:-}"
    exit 0
fi

if [[ -z "${COMPONENT:-}" || -z "${ISSUE:-}" ]]; then
    echo "Either COMPONENT or ISSUE has not been set, please ensure both are set."
    exit 0
fi

COMPONENT_PATH=$(resolve_component_path "${COMPONENT}")

if [[ -z "${COMPONENT_PATH}" ]]; then
    exit 0
fi

OWNERS=$(COMPONENT="${COMPONENT_PATH}" bash "${CUR_DIRECTORY}/get-codeowners.sh")

if [[ -z "${OWNERS}" ]]; then
    exit 0
fi

gh issue comment "${ISSUE}" --body "Pinging code owners for ${COMPONENT}: ${OWNERS}."
