#!/bin/bash

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Get the repository root directory (parent of tools directory)
SCRIPT_DIR=$(dirname "$0")
REPO_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
TOOLS_DIR="$REPO_ROOT/.tools"

GOPATH=$(go env GOPATH)
if [ -z "${GOPATH}" ] ; then
	echo "GOPATH is not defined."
	exit -1
fi

if [ ! -d "${GOPATH}" ] ; then
	echo "GOPATH ${GOPATH} is invalid"
	exit -1
fi

# Change to repository root for the rest of the operations
cd "$REPO_ROOT"

# Pre-requisites
if ! git diff --quiet; then \
	git status
	echo ""
	echo "Error: working tree is not clean"
	exit -1
fi


if [ "$(git tag --contains $(git log -1 --pretty=format:"%H"))" = "" ] ; then
	echo "$(git log -1)"
	echo ""
	echo "Error: HEAD is not pointing to a tagged version"
fi


make ${TOOLS_DIR}/gojq

DIR_TMP="${GOPATH}/src/oteltmp/"
rm -rf $DIR_TMP
mkdir -p $DIR_TMP

echo "Copy examples to ${DIR_TMP}"
cp -a ./examples ${DIR_TMP}

DIR_TMP="${GOPATH}/src/oteltmp/"
rm -rf $DIR_TMP
mkdir -p $DIR_TMP

echo "Copy examples to ${DIR_TMP}"
cp -a ./examples ${DIR_TMP}

# Update go.mod files
echo "Update go.mod: rename module and remove replace"
PACKAGE_DIRS=$(find . -mindepth 2 -type f -name 'go.mod' -exec dirname {} \; | egrep 'examples' | sed 's/^\.\///' | sort)

for dir in $PACKAGE_DIRS; do
	echo "  Update go.mod for $dir"
	
	# Check if the directory exists in the temporary location
	if [ ! -d "${DIR_TMP}/${dir}" ]; then
		echo "    Skipping $dir - directory not found in temporary location"
		continue
	fi
	
	(cd "${DIR_TMP}/${dir}" && \
	 # Get replaces, handle case where there are no replaces (returns null)
	 replaces_json=$(go mod edit -json | ${TOOLS_DIR}/gojq '.Replace // []' 2>/dev/null || echo '[]')
	 replaces=($(echo "$replaces_json" | ${TOOLS_DIR}/gojq -r '.[].Old.Path' 2>/dev/null || true))
	 
	 # Only process if there are actual replaces
	 if [ ${#replaces[@]} -gt 0 ] && [ "${replaces[0]}" != "" ]; then
		 # make an array (-dropreplace=mod1 -dropreplace=mod2 …)
		 dropreplaces=("${replaces[@]/#/-dropreplace=}")
		 go mod edit -module "oteltmp/${dir}" "${dropreplaces[@]}"
	 else
		 go mod edit -module "oteltmp/${dir}"
	 fi
	 
	 go mod tidy)
done

echo "Update done:"
echo ""

# Build directories that contain main package. These directories are different than
# directories that contain go.mod files.
echo "Build examples:"
EXAMPLES=$(find ./examples -type f -name go.mod -exec dirname {} \;)
for ex in $EXAMPLES; do
	echo "  Build $ex in ${DIR_TMP}/${ex}"
	(cd "${DIR_TMP}/${ex}" && \
	 go build .)
done

# Cleanup
echo "Remove copied files."
rm -rf $DIR_TMP
