// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

var validDecodingTests = []struct {
	Name     string
	Severity Severity
	Text     string
}{
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

	// Use the default SeverityInfo for an empty name.
	{"Default", SeverityInfo, ""},

	// Test case insensitivity.
	{"SeverityTraceLower", SeverityTrace1, "trace"},
	{"SeverityDebugMixed", SeverityDebug1, "Debug"},
	{"SeverityInfoMixed", SeverityInfo1, "InFo"},
	{"SeverityInfo3Lower", SeverityInfo3, "info3"},

	// Test offset calculations.
	{"SeverityTraceMinus2", SeverityTrace1 - 2, "TRACE-2"},
	{"SeverityWarnPlus2", SeverityWarn3, "WARN+2"},
	{"SeverityWarn2Plus2", SeverityWarn4, "WARN2+2"},
	{"SeverityErrorMinus4", SeverityWarn1, "ERROR-4"},
	{"SeverityError2Minus4", SeverityWarn2, "ERROR2-4"},
	{"SeverityFatalPlus10", SeverityFatal1 + 10, "FATAL+10"},

	// Test oversized fine-grained severity.
	{"SeverityTrace15", SeverityWarn3, "TRACE15"},
	{"SeverityTrace101", SeverityTrace1 + 100, "TRACE101"},

	// Test fine-grained severity of zero.
	{"SeverityTrace0", SeverityTrace, "TRACE0"},
	{"SeverityTrace0Plus1", SeverityTrace2, "TRACE0+1"},
}

var invalidText = []string{
	"UNKNOWN",
	"DEBUG3+abc",
	"INFO+abc",
	"ERROR-xyz",
	"not-a-level",
	"+1",
	"2",
	"2+1",
	"-1",
}

func TestSeverityString(t *testing.T) {
	for _, test := range validEncodingTests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Text, test.Severity.String())
		})
	}
}

func TestSeverityMarshalJSON(t *testing.T) {
	for _, test := range validEncodingTests {
		t.Run(test.Name, func(t *testing.T) {
			got, err := json.Marshal(test.Severity)
			require.NoError(t, err)
			assert.Equal(t, `"`+test.Text+`"`, string(got))
		})
	}
}

func TestSeverityUnmarshalJSON(t *testing.T) {
	for _, test := range validDecodingTests {
		t.Run(test.Name, func(t *testing.T) {
			var sev Severity
			data := []byte(`"` + test.Text + `"`)
			require.NoError(t, sev.UnmarshalJSON(data))
			const msg = "UnmarshalJSON(%q) != %d (%[2]s)"
			assert.Equalf(t, test.Severity, sev, msg, data, test.Severity)
		})
	}
}

func TestSeverityUnmarshalJSONError(t *testing.T) {
	invalidJSON := []string{
		`"UNKNOWN"`,
		`"DEBUG3+abc"`,
		`"INFO+abc"`,
		`"ERROR-xyz"`,
		`"not-a-level"`,
		`invalid-json`,
		`42`, // number instead of string
	}

	for _, test := range invalidJSON {
		t.Run(test, func(t *testing.T) {
			var sev Severity
			err := sev.UnmarshalJSON([]byte(test))
			assert.Error(t, err)
		})
	}
}

func TestSeverityMarshalText(t *testing.T) {
	for _, test := range validEncodingTests {
		t.Run(test.Name, func(t *testing.T) {
			got, err := test.Severity.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, test.Text, string(got))
		})
	}
}

func TestSeverityUnmarshalText(t *testing.T) {
	for _, test := range validDecodingTests {
		t.Run(test.Name, func(t *testing.T) {
			var sev Severity
			require.NoError(t, sev.UnmarshalText([]byte(test.Text)))
			const msg = "UnmarshalText(%q) != %d (%[2]s)"
			assert.Equalf(t, test.Severity, sev, msg, test.Text, test.Severity)
		})
	}
}

func TestSeverityUnmarshalTextError(t *testing.T) {
	for _, test := range invalidText {
		t.Run(test, func(t *testing.T) {
			var sev Severity
			err := sev.UnmarshalText([]byte(test))
			assert.Error(t, err)
		})
	}
}

func TestSeverityAppendText(t *testing.T) {
	tests := []struct {
		sev      Severity
		prefix   string
		expected string
	}{
		{SeverityInfo1, "", "INFO"},
		{SeverityError1, "level=", "level=ERROR"},
		{SeverityWarn3, "severity:", "severity:WARN3"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result, err := test.sev.AppendText([]byte(test.prefix))
			require.NoError(t, err)
			assert.Equal(t, test.expected, string(result))
		})
	}
}

func TestSeverityVarString(t *testing.T) {
	for _, test := range validEncodingTests {
		t.Run(test.Name, func(t *testing.T) {
			var sev SeverityVar
			sev.Set(test.Severity)

			want := "SeverityVar(" + test.Text + ")"
			assert.Equal(t, want, sev.String())
		})
	}
}

func TestSeverityVarMarshalText(t *testing.T) {
	for _, test := range validEncodingTests {
		t.Run(test.Name, func(t *testing.T) {
			var sev SeverityVar
			sev.Set(test.Severity)
			got, err := sev.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, test.Text, string(got))
		})
	}
}

func TestSeverityVarUnmarshalText(t *testing.T) {
	for _, test := range validDecodingTests {
		t.Run(test.Name, func(t *testing.T) {
			var sev SeverityVar
			require.NoError(t, sev.UnmarshalText([]byte(test.Text)))

			got := Severity(int(sev.val.Load()))
			const msg = "UnmarshalText(%q) != %d (%[2]s)"
			assert.Equalf(t, test.Severity, got, msg, test.Text, test.Severity)
		})
	}
}

func TestSeverityVarUnmarshalTextError(t *testing.T) {
	for _, test := range invalidText {
		t.Run(test, func(t *testing.T) {
			var sev SeverityVar
			err := sev.UnmarshalText([]byte(test))
			assert.Error(t, err)
		})
	}
}

func TestSeverityVarAppendText(t *testing.T) {
	tests := []struct {
		sev      Severity
		prefix   string
		expected string
	}{
		{SeverityInfo1, "", "INFO"},
		{SeverityError1, "level=", "level=ERROR"},
		{SeverityWarn2, "severity:", "severity:WARN2"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			var sev SeverityVar
			sev.Set(test.sev)
			result, err := sev.AppendText([]byte(test.prefix))
			require.NoError(t, err)
			assert.Equal(t, test.expected, string(result))
		})
	}
}

func TestSeveritySeverityClamps(t *testing.T) {
	t.Run("BelowRange", func(t *testing.T) {
		got := (SeverityTrace1 - 10).Severity()
		assert.Equal(t, log.SeverityTrace1, got)
	})
	t.Run("AboveRange", func(t *testing.T) {
		got := (SeverityFatal4 + 10).Severity()
		assert.Equal(t, log.SeverityFatal4, got)
	})
	t.Run("WithinRange", func(t *testing.T) {
		// Explicit table to verify each defined severity (including aliases) maps
		// to the expected log.Severity. This guards against accidental reorder or
		// gaps because expectations are enumerated instead of derived.
		tests := []struct {
			name string
			sev  Severity
			want log.Severity
		}{
			// Aliases (base names) first.
			{"Alias/SeverityTrace", SeverityTrace, log.SeverityTrace1},
			{"Alias/SeverityDebug", SeverityDebug, log.SeverityDebug1},
			{"Alias/SeverityInfo", SeverityInfo, log.SeverityInfo1},
			{"Alias/SeverityWarn", SeverityWarn, log.SeverityWarn1},
			{"Alias/SeverityError", SeverityError, log.SeverityError1},
			{"Alias/SeverityFatal", SeverityFatal, log.SeverityFatal1},
			// Full set of defined granular severities.
			{"SeverityTrace1", SeverityTrace1, log.SeverityTrace1},
			{"SeverityTrace2", SeverityTrace2, log.SeverityTrace2},
			{"SeverityTrace3", SeverityTrace3, log.SeverityTrace3},
			{"SeverityTrace4", SeverityTrace4, log.SeverityTrace4},
			{"SeverityDebug1", SeverityDebug1, log.SeverityDebug1},
			{"SeverityDebug2", SeverityDebug2, log.SeverityDebug2},
			{"SeverityDebug3", SeverityDebug3, log.SeverityDebug3},
			{"SeverityDebug4", SeverityDebug4, log.SeverityDebug4},
			{"SeverityInfo1", SeverityInfo1, log.SeverityInfo1},
			{"SeverityInfo2", SeverityInfo2, log.SeverityInfo2},
			{"SeverityInfo3", SeverityInfo3, log.SeverityInfo3},
			{"SeverityInfo4", SeverityInfo4, log.SeverityInfo4},
			{"SeverityWarn1", SeverityWarn1, log.SeverityWarn1},
			{"SeverityWarn2", SeverityWarn2, log.SeverityWarn2},
			{"SeverityWarn3", SeverityWarn3, log.SeverityWarn3},
			{"SeverityWarn4", SeverityWarn4, log.SeverityWarn4},
			{"SeverityError1", SeverityError1, log.SeverityError1},
			{"SeverityError2", SeverityError2, log.SeverityError2},
			{"SeverityError3", SeverityError3, log.SeverityError3},
			{"SeverityError4", SeverityError4, log.SeverityError4},
			{"SeverityFatal1", SeverityFatal1, log.SeverityFatal1},
			{"SeverityFatal2", SeverityFatal2, log.SeverityFatal2},
			{"SeverityFatal3", SeverityFatal3, log.SeverityFatal3},
			{"SeverityFatal4", SeverityFatal4, log.SeverityFatal4},
		}
		for _, tc := range tests {
			assert.Equalf(t, tc.want, tc.sev.Severity(), tc.name)
		}
	})
}

// Test JSON roundtrip for structures containing Severity.
func TestSeverityJSONRoundtrip(t *testing.T) {
	type Config struct {
		Level Severity `json:"level"`
		Name  string   `json:"name"`
	}

	original := Config{
		Level: SeverityError1,
		Name:  "test-config",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	expectedJSON := `{"level":"ERROR","name":"test-config"}`
	assert.JSONEq(t, expectedJSON, string(data))

	// Unmarshal from JSON
	var decoded Config
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

// Test text marshaling roundtrip for SeverityVar.
func TestSeverityVarTextRoundtrip(t *testing.T) {
	original := SeverityWarn3

	var sev SeverityVar
	sev.Set(original)

	// Marshal to text.
	data, err := sev.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "WARN3", string(data))

	// Unmarshal from text
	var decoded SeverityVar
	require.NoError(t, decoded.UnmarshalText(data))
	assert.Equal(t, original, Severity(int(decoded.val.Load())))
}
