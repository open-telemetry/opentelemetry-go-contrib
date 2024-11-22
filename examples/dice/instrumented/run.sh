# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

#!/bin/bash

go mod tidy
export OTEL_RESOURCE_ATTRIBUTES="service.name=dice,service.version=0.1.0"
go run .