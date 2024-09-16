// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev // import "go.opentelemetry.io/contrib/processors/minsev"

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	api "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
)

const key = "OTEL_LOG_LEVEL"

var getSeverity = sync.OnceValue(func() api.Severity {
	conv := map[string]api.Severity{
		"":      api.SeverityInfo, // Default to SeverityInfo for unset.
		"debug": api.SeverityDebug,
		"info":  api.SeverityInfo,
		"warn":  api.SeverityWarn,
		"error": api.SeverityError,
	}
	// log.SeverityUnknown for unknown values.
	return conv[strings.ToLower(os.Getenv(key))]
})

type EnvSeverity struct{}

func (EnvSeverity) Severity() api.Severity { return getSeverity() }

func ExampleSeveritier() {
	// Mock an environment variable setup that would be done externally.
	_ = os.Setenv(key, "error")

	p := NewLogProcessor(&processor{}, EnvSeverity{})

	ctx, r := context.Background(), log.Record{}
	r.SetSeverity(api.SeverityDebug)
	fmt.Println(p.Enabled(ctx, r))

	r.SetSeverity(api.SeverityError)
	fmt.Println(p.Enabled(ctx, r))

	// Output:
	// false
	// true
}
