#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#

set -euo pipefail

SCRIPT_DIR=$(dirname "$0")
CODEOWNERS="${SCRIPT_DIR}/../CODEOWNERS"

declare -a cases=(
    'detector: aws:ec2|detectors/aws/ec2/v2'
    'detector: aws:ecs|detectors/aws/ecs'
    'detector: aws:eks|detectors/aws/eks'
    'detector: aws:lambda|detectors/aws/lambda'
    'detector: gcp|detectors/gcp'
    'instrumentation: host|instrumentation/host'
    'instrumentation: otelaws|instrumentation/github.com/aws/aws-sdk-go-v2/otelaws'
    'instrumentation: otelgin|instrumentation/github.com/gin-gonic/gin/otelgin'
    'instrumentation: otelgrpc|instrumentation/google.golang.org/grpc/otelgrpc'
    'instrumentation: otelhttp|instrumentation/net/http/otelhttp'
    'instrumentation: otelhttptrace|instrumentation/net/http/httptrace/otelhttptrace'
    'instrumentation: otellambda|instrumentation/github.com/aws/aws-lambda-go/otellambda'
    'instrumentation: otelmongo|instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo'
    'instrumentation: otelmux|instrumentation/github.com/gorilla/mux/otelmux'
    'instrumentation: otelrestful|instrumentation/github.com/emicklei/go-restful/otelrestful'
    'instrumentation: runtime|instrumentation/runtime'
    'propagator: Autoprop|propagators/autoprop'
    'propagator: AWS X-Ray|propagators/aws'
    'propagator: B3|propagators/b3'
    'propagator: Jaeger|propagators/jaeger'
    'propagator: OpenCensus|propagators/opencensus'
    'propagator: OT|propagators/ot'
    'sampler: JaegerRemote|samplers/jaegerremote'
    'sampler: probability:consistent|samplers/probability/consistent'
    'exporter: exporters/autoexport|exporters/autoexport'
    'zpages: zpages|zpages'
)

# These CODEOWNERS paths are not selectable in the issue form. Keep this list
# explicit so adding a new CODEOWNERS entry requires an intentional mapping or
# an intentional exclusion from issue component labels.
declare -a non_issue_components=(
    'bridges/otellogr'
    'bridges/otellogrus'
    'bridges/otelslog'
    'bridges/otelzap'
    'bridges/prometheus'
    'detectors/autodetect'
    'detectors/aws/elasticbeanstalk'
    'detectors/azure'
    'detectors/azure/azureappservice'
    'detectors/azure/azurecontainerapps'
    'detectors/azure/azurefunctions'
    'detectors/azure/azurevm'
    'detectors/docker'
    'detectors/hetzner'
    'detectors/ibmcloud/vpc'
    'detectors/k8sapi'
    'detectors/vultr'
    'instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo'
    'otelconf'
    'processors/baggagecopy'
    'processors/minsev'
    'propagators/envcar'
)

for test_case in "${cases[@]}"; do
    component=${test_case%%|*}
    expected_path=${test_case#*|}
    actual_path=$(bash "${SCRIPT_DIR}/ping_codeowners_issues.sh" --resolve "${component}")
    [[ "${actual_path}" == "${expected_path}" ]] || {
        echo "${component}: expected ${expected_path}, got ${actual_path}" >&2
        exit 1
    }
    awk -v expected_path="${expected_path}" '$1 !~ /^#/ { path = $1; sub(/\/$/, "", path); if (path == expected_path) found = 1 } END { exit !found }' "${CODEOWNERS}" || {
        echo "${component}: ${expected_path} is missing from CODEOWNERS" >&2
        exit 1
    }
done

codeowners_paths=$(awk '$1 !~ /^#/ && NF >= 2 { path = $1; sub(/\/$/, "", path); if (path != "*" && path != "CODEOWNERS") print path }' "${CODEOWNERS}" | LC_ALL=C sort -u)
mapped_paths=$(printf '%s\n' "${cases[@]}" | cut -d'|' -f2 | LC_ALL=C sort -u)
excluded_paths=$(printf '%s\n' "${non_issue_components[@]}" | LC_ALL=C sort -u)
unexpected_paths=$(comm -23 <(printf '%s\n' "${codeowners_paths}") <(printf '%s\n' "${mapped_paths}" "${excluded_paths}" | LC_ALL=C sort -u))
missing_codeowners_paths=$(comm -13 <(printf '%s\n' "${codeowners_paths}") <(printf '%s\n' "${mapped_paths}" | LC_ALL=C sort -u))
stale_exclusions=$(comm -23 <(printf '%s\n' "${excluded_paths}") <(printf '%s\n' "${codeowners_paths}"))

if [[ -n "${unexpected_paths}" ]]; then
    echo "CODEOWNERS paths need an issue mapping or explicit exclusion:" >&2
    printf '%s\n' "${unexpected_paths}" >&2
    exit 1
fi

if [[ -n "${missing_codeowners_paths}" ]]; then
    echo "Resolver mappings point to paths missing from CODEOWNERS:" >&2
    printf '%s\n' "${missing_codeowners_paths}" >&2
    exit 1
fi

if [[ -n "${stale_exclusions}" ]]; then
    echo "Non-issue component exclusions are missing from CODEOWNERS:" >&2
    printf '%s\n' "${stale_exclusions}" >&2
    exit 1
fi

normalize_component_label() {
    case "$1" in
        AutoExporter) echo "exporter: exporters/autoexport" ;;
        Config) return 1 ;;
        zPages) echo "zpages: zpages" ;;
        *)
            local component_type component_name
            component_type=$(printf '%s' "$1" | cut -d: -f1 | tr '[:upper:]' '[:lower:]')
            component_name=$(printf '%s' "$1" | cut -d: -f2- | sed 's/^ //' | tr '[:upper:]' '[:lower:]' | sed 's/[[:space:]]\+/:/g; s/x-ray/xray/g')
            echo "${component_type}: ${component_name}"
            ;;
    esac
}

for template in "${SCRIPT_DIR}/../.github/ISSUE_TEMPLATE/bug_report.yaml" "${SCRIPT_DIR}/../.github/ISSUE_TEMPLATE/feature_request.yaml"; do
    while IFS= read -r form_value; do
        label=$(normalize_component_label "${form_value}") || continue
        resolved_path=$(bash "${SCRIPT_DIR}/ping_codeowners_issues.sh" --resolve "${label}")
        [[ -n "${resolved_path}" ]] || {
            echo "${template}: ${form_value} has no component mapping" >&2
            exit 1
        }
        awk -v expected_path="${resolved_path}" '$1 !~ /^#/ { path = $1; sub(/\/$/, "", path); if (path == expected_path) found = 1 } END { exit !found }' "${CODEOWNERS}" || {
            echo "${template}: ${form_value} maps to missing CODEOWNERS path ${resolved_path}" >&2
            exit 1
        }
    done < <(awk '
        /id: otel_component/ { in_component = 1 }
        in_component && /options:/ { in_options = 1; next }
        in_options && /^[[:space:]]+- / {
            value = $0
            sub(/^[[:space:]]+-[[:space:]]*/, "", value)
            gsub(/^'"'"'|[[:space:]]*'"'"'$/, "", value)
            print value
            next
        }
        in_options && /^[[:space:]]+validations:/ { exit }
    ' "${template}")
done