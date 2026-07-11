// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package host

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePSI(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		validate    func(*testing.T, *psiStats)
	}{
		{
			name: "valid cpu format",
			input: `some avg10=1.23 avg60=2.34 avg300=3.45 total=123456
`,
			expectError: false,
			validate: func(t *testing.T, stats *psiStats) {
				assert.Equal(t, 1.23, stats.some.avg10)
				assert.Equal(t, 2.34, stats.some.avg60)
				assert.Equal(t, 3.45, stats.some.avg300)
				assert.Equal(t, int64(123456), stats.some.total)
				assert.Equal(t, 0.0, stats.full.avg10)
				assert.Equal(t, 0.0, stats.full.avg60)
				assert.Equal(t, 0.0, stats.full.avg300)
				assert.Equal(t, int64(0), stats.full.total)
			},
		},
		{
			name: "valid memory format with full",
			input: `some avg10=1.23 avg60=2.34 avg300=3.45 total=123456
full avg10=0.50 avg60=1.00 avg300=1.50 total=654321
`,
			expectError: false,
			validate: func(t *testing.T, stats *psiStats) {
				assert.Equal(t, 1.23, stats.some.avg10)
				assert.Equal(t, 2.34, stats.some.avg60)
				assert.Equal(t, 3.45, stats.some.avg300)
				assert.Equal(t, int64(123456), stats.some.total)
				assert.Equal(t, 0.50, stats.full.avg10)
				assert.Equal(t, 1.00, stats.full.avg60)
				assert.Equal(t, 1.50, stats.full.avg300)
				assert.Equal(t, int64(654321), stats.full.total)
			},
		},
		{
			name: "zero values",
			input: `some avg10=0.00 avg60=0.00 avg300=0.00 total=0
full avg10=0.00 avg60=0.00 avg300=0.00 total=0
`,
			expectError: false,
			validate: func(t *testing.T, stats *psiStats) {
				assert.Equal(t, 0.0, stats.some.avg10)
				assert.Equal(t, 0.0, stats.some.avg60)
				assert.Equal(t, 0.0, stats.some.avg300)
				assert.Equal(t, int64(0), stats.some.total)
				assert.Equal(t, 0.0, stats.full.avg10)
				assert.Equal(t, 0.0, stats.full.avg60)
				assert.Equal(t, 0.0, stats.full.avg300)
				assert.Equal(t, int64(0), stats.full.total)
			},
		},
		{
			name: "large values",
			input: `some avg10=99.99 avg60=100.00 avg300=50.00 total=9223372036854775807
full avg10=25.50 avg60=30.00 avg300=35.00 total=1234567890123456
`,
			expectError: false,
			validate: func(t *testing.T, stats *psiStats) {
				assert.Equal(t, 99.99, stats.some.avg10)
				assert.Equal(t, 100.00, stats.some.avg60)
				assert.Equal(t, 50.00, stats.some.avg300)
				assert.Equal(t, int64(9223372036854775807), stats.some.total)
				assert.Equal(t, 25.50, stats.full.avg10)
				assert.Equal(t, 30.00, stats.full.avg60)
				assert.Equal(t, 35.00, stats.full.avg300)
				assert.Equal(t, int64(1234567890123456), stats.full.total)
			},
		},
		{
			name:        "invalid format - not enough fields",
			input:       "some avg10=1.23\n",
			expectError: true,
		},
		{
			name:        "invalid format - bad avg10 value",
			input:       "some avg10=abc avg60=2.34 avg300=3.45 total=123456\n",
			expectError: true,
		},
		{
			name:        "invalid format - bad avg60 value",
			input:       "some avg10=1.23 avg60=xyz avg300=3.45 total=123456\n",
			expectError: true,
		},
		{
			name:        "invalid format - bad avg300 value",
			input:       "some avg10=1.23 avg60=2.34 avg300=bad total=123456\n",
			expectError: true,
		},
		{
			name:        "invalid format - bad total value",
			input:       "some avg10=1.23 avg60=2.34 avg300=3.45 total=abc\n",
			expectError: true,
		},
		{
			name:        "invalid pressure type",
			input:       "partial avg10=1.23 avg60=2.34 avg300=3.45 total=123456\n",
			expectError: true,
		},
		{
			name:        "empty input",
			input:       "",
			expectError: false,
			validate: func(t *testing.T, stats *psiStats) {
				// Empty input should return zero values
				assert.Equal(t, 0.0, stats.some.avg10)
				assert.Equal(t, 0.0, stats.full.avg10)
			},
		},
		{
			name: "only whitespace",
			input: `

`,
			expectError: false,
			validate: func(t *testing.T, stats *psiStats) {
				assert.Equal(t, 0.0, stats.some.avg10)
				assert.Equal(t, 0.0, stats.full.avg10)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stats, err := parsePSI(tc.input)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, stats)
				}
			}
		})
	}
}

func TestParsePSI_RealWorldExamples(t *testing.T) {
	// Example from a real Linux system under light load
	lightLoad := `some avg10=0.05 avg60=0.12 avg300=0.08 total=1234567
full avg10=0.01 avg60=0.02 avg300=0.03 total=234567
`
	stats, err := parsePSI(lightLoad)
	require.NoError(t, err)
	assert.Equal(t, 0.05, stats.some.avg10)
	assert.Equal(t, 0.01, stats.full.avg10)

	// Example from a system with no pressure (CPU typically only has "some")
	noPressure := `some avg10=0.00 avg60=0.00 avg300=0.00 total=0
`
	stats, err = parsePSI(noPressure)
	require.NoError(t, err)
	assert.Equal(t, 0.0, stats.some.avg10)
	assert.Equal(t, int64(0), stats.some.total)
}
