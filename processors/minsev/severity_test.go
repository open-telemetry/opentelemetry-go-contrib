// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/log"
)

func TestSeverityVarConcurrentSafe(*testing.T) {
	var (
		sev SeverityVar
		wg  sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for s := SeverityTrace1; s <= SeverityFatal4; s++ {
			sev.Set(s)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var got log.Severity
		for i := SeverityFatal4 - SeverityTrace1; i >= 0; i-- {
			got = sev.Severity()
		}
		_ = got
	}()

	wg.Wait()
}

var validEncodingTests = []struct {
	Name     string
	Severity Severity
	Text     string
}{
	// Use offset for values less than SeverityTrace1.
	{"SeverityTraceMinus2", SeverityTrace - 2, "TRACE-2"},

	{"SeverityTrace", SeverityTrace, "TRACE"},
	{"SeverityTrace1", SeverityTrace1, "TRACE"},
	{"SeverityTrace2", SeverityTrace2, "TRACE2"},
	{"SeverityTrace3", SeverityTrace3, "TRACE3"},
	{"SeverityTrace4", SeverityTrace4, "TRACE4"},
	{"SeverityDebug", SeverityDebug, "DEBUG"},
	{"SeverityDebug1", SeverityDebug1, "DEBUG"},
	{"SeverityDebug2", SeverityDebug2, "DEBUG2"},
	{"SeverityDebug3", SeverityDebug3, "DEBUG3"},
	{"SeverityDebug4", SeverityDebug4, "DEBUG4"},
	{"SeverityInfo", SeverityInfo, "INFO"},
	{"SeverityInfo1", SeverityInfo1, "INFO"},
	{"SeverityInfo2", SeverityInfo2, "INFO2"},
	{"SeverityInfo3", SeverityInfo3, "INFO3"},
	{"SeverityInfo4", SeverityInfo4, "INFO4"},
	{"SeverityWarn", SeverityWarn, "WARN"},
	{"SeverityWarn1", SeverityWarn1, "WARN"},
	{"SeverityWarn2", SeverityWarn2, "WARN2"},
	{"SeverityWarn3", SeverityWarn3, "WARN3"},
	{"SeverityWarn4", SeverityWarn4, "WARN4"},
	{"SeverityError", SeverityError, "ERROR"},
	{"SeverityError1", SeverityError1, "ERROR"},
	{"SeverityError2", SeverityError2, "ERROR2"},
	{"SeverityError3", SeverityError3, "ERROR3"},
	{"SeverityError4", SeverityError4, "ERROR4"},
	{"SeverityFatal", SeverityFatal, "FATAL"},
	{"SeverityFatal1", SeverityFatal1, "FATAL"},
	{"SeverityFatal2", SeverityFatal2, "FATAL2"},
	{"SeverityFatal3", SeverityFatal3, "FATAL3"},
	{"SeverityFatal4", SeverityFatal4, "FATAL4"},

	// Use offset for values greater than SeverityFatal4.
	{"SeverityFatal4Plus2", SeverityFatal4 + 2, "FATAL+6"},
}

func TestSeverityString(t *testing.T) {
	for _, test := range validEncodingTests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Text, test.Severity.String())
		})
	}
}
