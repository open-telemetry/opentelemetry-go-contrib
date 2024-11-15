// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/trace"
)

func TestTraceIDIsValidLength(t *testing.T) {
	idg := NewIDGenerator()
	traceID, _ := idg.NewIDs(context.Background())

	expectedTraceIDLength := 32
	assert.Len(t, traceID.String(), expectedTraceIDLength, "TraceID has incorrect length.")
}

func TestTraceIDIsUnique(t *testing.T) {
	idg := NewIDGenerator()
	traceID1, _ := idg.NewIDs(context.Background())
	traceID2, _ := idg.NewIDs(context.Background())

	assert.NotEqual(t, traceID1.String(), traceID2.String(), "TraceID should be unique")
}

func TestTraceIDTimestampInBounds(t *testing.T) {
	idg := NewIDGenerator()

	previousTime := time.Now().Unix()

	traceID, _ := idg.NewIDs(context.Background())

	currentTime, err := strconv.ParseInt(traceID.String()[0:8], 16, 64)
	require.NoError(t, err)

	nextTime := time.Now().Unix()

	assert.LessOrEqual(t, previousTime, currentTime, "TraceID is generated incorrectly with the wrong timestamp.")
	assert.LessOrEqual(t, currentTime, nextTime, "TraceID is generated incorrectly with the wrong timestamp.")
}

func TestTraceIDIsNotNil(t *testing.T) {
	var nilTraceID trace.TraceID
	idg := NewIDGenerator()
	traceID, _ := idg.NewIDs(context.Background())

	assert.False(t, bytes.Equal(traceID[:], nilTraceID[:]), "TraceID cannot be empty.")
}

func TestSpanIDIsValidLength(t *testing.T) {
	idg := NewIDGenerator()
	ctx := context.Background()
	traceID, spanID1 := idg.NewIDs(ctx)
	spanID2 := idg.NewSpanID(context.Background(), traceID)
	expectedSpanIDLength := 16

	assert.Len(t, spanID1.String(), expectedSpanIDLength, "SpanID has incorrect length")
	assert.Len(t, spanID2.String(), expectedSpanIDLength, "SpanID has incorrect length")
}

func TestSpanIDIsUnique(t *testing.T) {
	idg := NewIDGenerator()
	ctx := context.Background()
	traceID, spanID1 := idg.NewIDs(ctx)
	_, spanID2 := idg.NewIDs(ctx)

	spanID3 := idg.NewSpanID(ctx, traceID)
	spanID4 := idg.NewSpanID(ctx, traceID)

	assert.NotEqual(t, spanID1.String(), spanID2.String(), "SpanID should be unique")
	assert.NotEqual(t, spanID3.String(), spanID4.String(), "SpanID should be unique")
}

func TestSpanIDIsNotNil(t *testing.T) {
	var nilSpanID trace.SpanID
	idg := NewIDGenerator()
	ctx := context.Background()
	traceID, spanID1 := idg.NewIDs(ctx)
	spanID2 := idg.NewSpanID(ctx, traceID)

	assert.False(t, bytes.Equal(spanID1[:], nilSpanID[:]), "SpanID cannot be empty.")
	assert.False(t, bytes.Equal(spanID2[:], nilSpanID[:]), "SpanID cannot be empty.")
}
