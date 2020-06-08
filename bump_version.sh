#!/usr/bin/env bash

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

#
# This script is used for
# a) creating a new tagged release of go.opentelemetry.io/contrib
# b) bumping the referenced version of go.opentelemetry.io/otel
#
# The options can be used together or individually.
#
set -e

help() {
   printf "\n"
   printf "Usage: %s [-o otel_tag] [-t tag]\n" "$0"
   printf "\t-o Otel release tag. Update all go.mod to reference go.opentelemetry.io/otel <otel_tag>.\n"
   printf "\t-t New unreleased tag. Update all go.mod with this tag.\n"
   exit 1 # Exit script after printing help
}

while getopts "t:o:" opt
do
   case "$opt" in
      t ) TAG="$OPTARG" ;;
      o ) OTEL_TAG="$OPTARG" ;;
      ? ) help ;; # Print help
   esac
done

declare -r SEMVER_REGEX="^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(\\-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

validate_tag() {
    local tag_=$1
    if [[ "${tag_}" =~ ${SEMVER_REGEX} ]]; then
	    printf "%s is valid semver tag.\n" "${tag_}"
    else
	    printf "%s is not a valid semver tag.\n" "${tag_}"
	    return 1
    fi
}

# Print help in case parameters are empty
if [[ -z "$TAG" && -z "$OTEL_TAG" ]]
then
    printf "At least one of '-o' or '-t' must be specified.\n"
    help
fi


## Validate tags first
if [ -n "${OTEL_TAG}" ]; then
    validate_tag "${OTEL_TAG}" || exit $?
    # check that OTEL_TAG is a currently released tag for go.opentelemetry.io/otel
    TMPDIR=$(mktemp -d "/tmp/otel-contrib.XXXXXX") || exit 1
    trap "rm -fr ${TMPDIR}" EXIT
    (cd "${TMPDIR}" && go mod init tagtest)
    # requires go 1.14 for support of '-modfile'
    if ! go get -modfile="${TMPDIR}/go.mod" -d -v "go.opentelemetry.io/otel@${OTEL_TAG}"; then
        printf "go.opentelemetry.io/otel %s does not exist. Please supply a valid tag\n" "${OTEL_TAG}"
        exit 1
    fi
fi
if [ -n "${TAG}" ]; then
    validate_tag "${TAG}" || exit $?
    TAG_FOUND=$(git tag --list "${TAG}")
    if [[ ${TAG_FOUND} = "${TAG}" ]] ; then
        printf "Tag %s already exists\n" "${TAG}"
        exit 1
    fi
fi

cd "$(dirname "$0")"

if ! git diff --quiet; then \
    printf "Working tree is not clean, can't proceed\n"
    git status
    git diff
    exit 1
fi

declare BRANCH_NAME=pre_release_${TAG}
if [ -z "${TAG}" ]; then
    BRANCH_NAME=bump_otel_${OTEL_TAG}
fi

patch_gomods() {
    local pkg_=$1
    local tag_=$2
    # now do the same for all the directories underneath
    PACKAGE_DIRS=$(find . -mindepth 2 -type f -name 'go.mod' -exec dirname {} \; | egrep -v 'tools' | sed 's|^\.\/||' | sort)
    # quote any '.' characters in the pkg name
    local quoted_pkg_=${pkg_//./\\.}
    for dir in $PACKAGE_DIRS; do
	    cp "${dir}/go.mod" "${dir}/go.mod.bak"
	    sed "s|${quoted_pkg_}\([^ ]*\) v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*[^0-9]*.*$|${pkg_}\1 ${tag_}|" "${dir}/go.mod.bak" >"${dir}/go.mod"
	    rm -f "${dir}/go.mod.bak"
    done
}

# branch off from existing master
git checkout -b "${BRANCH_NAME}" master

# Update go.mods
if [ -n "${OTEL_TAG}" ]; then
    # first update the top most module
    go get "go.opentelemetry.io/otel@${OTEL_TAG}"
    patch_gomods go.opentelemetry.io/otel "${OTEL_TAG}"
fi

if [ -n "${TAG}" ]; then
    patch_gomods go.opentelemetry.io/contrib "${TAG}"
fi

git diff
# Run lint to update go.sum
make lint

# Add changes and commit.
git add .
make ci
declare COMMIT_MSG="Prepare for releasing $TAG"
if [ -z "${TAG}" ]; then
    COMMIT_MSG="Bumping otel version to ${OTEL_TAG}"
fi
git commit -m "${COMMIT_MSG}"

printf "Now run following to verify the changes.\ngit diff master\n"
printf "\nThen push the changes to upstream\n"
