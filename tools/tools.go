// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools
// +build tools

package tools // import "go.opentelemetry.io/contrib/tools"

import (
	_ "github.com/atombender/go-jsonschema"
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/itchyny/gojq"
	_ "github.com/jcchavezs/porto/cmd/porto"
	_ "github.com/wadey/gocovmerge"
	_ "go.opentelemetry.io/build-tools/crosslink"
	_ "go.opentelemetry.io/build-tools/gotmpl"
	_ "go.opentelemetry.io/build-tools/multimod"
	_ "golang.org/x/exp/cmd/gorelease"
	_ "golang.org/x/tools/cmd/stringer"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
