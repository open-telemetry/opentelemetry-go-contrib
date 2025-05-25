#!/bin/bash

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Get the repository root directory (parent of tools directory)
SCRIPT_DIR=$(dirname "$0")
REPO_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
TOOLS_DIR="$REPO_ROOT/.tools"
TOOLS_MOD_DIR="$REPO_ROOT/tools"

GOPATH=$(go env GOPATH)
if [ -z "${GOPATH}" ] ; then
	printf "GOPATH is not defined.\n"
	exit -1
fi

if [ ! -d "${GOPATH}" ] ; then
	printf "GOPATH ${GOPATH} is invalid \n"
	exit -1
fi

# Pre-requisites
if ! git diff --quiet; then \
	git status
	printf "\n\nError: working tree is not clean\n"
	exit -1
fi

#if [ "$(git tag --contains $(git log -1 --pretty=format:"%H"))" = "" ] ; then
#	printf "$(git log -1)"
#	printf "\n\nError: HEAD is not pointing to a tagged version"
#fi


mkdir -p "${TOOLS_DIR}"
printf "Building gojq tool...\n"
(cd "${TOOLS_MOD_DIR}" && go build -o "${TOOLS_DIR}/gojq" github.com/itchyny/gojq/cmd/gojq)




DIR_TMP="${GOPATH}/src/oteltmp/"
rm -rf $DIR_TMP
mkdir -p $DIR_TMP

printf "Copy examples to ${DIR_TMP}\n"
cp -a ./examples ${DIR_TMP}

# Update go.mod files
printf "Update go.mod: rename module and remove replace\n"

PACKAGE_DIRS=$(find . -mindepth 2 -type f -name 'go.mod' -exec dirname {} \; | egrep 'examples' | sed 's/^\.\///' | sort)

for dir in $PACKAGE_DIRS; do
	printf "  Update go.mod for $dir\n"
	(cd "${DIR_TMP}/${dir}" && \
	 # replaces is ("mod1" "mod2" …)
	 replaces=($(go mod edit -json | ${TOOLS_DIR}/gojq '.Replace[].Old.Path' || true))
	 # strip double quotes
	 replaces=("${replaces[@]%\"}") && \
	 replaces=("${replaces[@]#\"}") && \
	 # make an array (-dropreplace=mod1 -dropreplace=mod2 …)
	 dropreplaces=("${replaces[@]/#/-dropreplace=}") && \
	 go mod edit -module "oteltmp/${dir}" "${dropreplaces[@]}" && \
	 go mod tidy)
done
printf "Update done:\n\n"

# Build directories that contain main package. These directories are different than
# directories that contain go.mod files.
printf "Build examples:\n"

EXAMPLES=$(find ./examples -type f -name go.mod -exec dirname {} \;)
for ex in $EXAMPLES; do
	printf "  Build $ex in ${DIR_TMP}/${ex}\n"
	(cd "${DIR_TMP}/${ex}" && \
	 go build .)
done

# Cleanup
printf "Remove copied files.\n"
rm -rf $DIR_TMP
