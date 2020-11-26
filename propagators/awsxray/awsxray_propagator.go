// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/api/trace"
)

const (
	traceHeaderKey       = "X-Amzn-Trace-Id"
	traceHeaderDelimiter = ";"
	kvDelimiter          = "="
	traceIDKey           = "Root"
	sampleFlagKey        = "Sampled"
	parentIDKey          = "Parent"
	traceIDVersion       = "1"
	traceIDDelimiter     = "-"
	isSampled            = "1"
	notSampled           = "0"

	traceFlagNone           = 0x0
	traceFlagSampled        = 0x1 << 0
	traceIDLength           = 35
	traceIDDelimitterIndex1 = 1
	traceIDDelimitterIndex2 = 10
	traceIDFirstPartLength  = 8
	sampledFlagLength       = 1
)

var (
	empty                  = trace.EmptySpanContext()
	errInvalidTraceHeader  = errors.New("invalid X-Amzn-Trace-Id header value, should contain 3 different part separated by ;")
	errMalformedTraceID    = errors.New("cannot decode trace id from header, should be a string of hex, lowercase trace id can't be all zero")
	errInvalidSpanIDLength = errors.New("invalid span id length, must be 16")
)

// AWS X-Ray propagator serializes Span Context to/from AWS X-Ray headers
//
// AWS X-Ray format
//
// X-Amzn-Trace-Id: Root={traceId};Parent={parentId};Sampled={samplingFlag}
type Xray struct{}

// Asserts that the propagator implements the otel.textMapPropagator interface
var _ otel.TextMapPropagator = &Xray{}

// Inject injects a context to the carrier following AWS X-Ray format.
func (awsxray Xray) Inject(ctx context.Context, carrier otel.TextMapCarrier) {
	sc := trace.SpanFromContext(ctx).SpanContext()
	headers := []string{}
	if !sc.TraceID.IsValid() || !sc.SpanID.IsValid() {
		return
	}
	otTraceID := sc.TraceID.String()
	xrayTraceID := traceIDVersion + traceIDDelimiter + otTraceID[0:traceIDFirstPartLength] +
		traceIDDelimiter + otTraceID[traceIDFirstPartLength:]
	parentID := sc.SpanID
	samplingFlag := notSampled
	if sc.TraceFlags == traceFlagSampled {
		samplingFlag = isSampled
	}

	headers = append(headers, traceIDKey, kvDelimiter, xrayTraceID, traceHeaderDelimiter, parentIDKey,
		kvDelimiter, parentID.String(), traceHeaderDelimiter, sampleFlagKey, kvDelimiter, samplingFlag)

	carrier.Set(traceHeaderKey, strings.Join(headers, ""))
}

// Extract gets a context from the carrier if it contains AWS X-Ray headers.
func (awsxray Xray) Extract(ctx context.Context, carrier otel.TextMapCarrier) context.Context {
	// extract tracing information
	if header := carrier.Get(traceHeaderKey); header != "" {
		sc, err := extract(header)
		if err == nil && sc.IsValid() {
			return trace.ContextWithRemoteSpanContext(ctx, sc)
		}
	}
	return ctx
}

// extracts Span Context from context
func extract(headerVal string) (trace.SpanContext, error) {
	var (
		sc             = trace.SpanContext{}
		err            error
		delimiterIndex int
		part           string
	)
	pos := 0
	for pos < len(headerVal) {
		delimiterIndex = indexOf(headerVal, traceHeaderDelimiter, pos)
		if delimiterIndex >= 0 {
			part = headerVal[pos:delimiterIndex]
			pos = delimiterIndex + 1
		} else {
			//last part
			part = strings.TrimSpace(headerVal[pos:])
			pos = len(headerVal)
		}
		equalsIndex := strings.Index(part, kvDelimiter)
		if equalsIndex < 0 {
			return empty, errInvalidTraceHeader
		}
		value := part[equalsIndex+1:]
		if strings.HasPrefix(part, traceIDKey) {
			sc.TraceID, err = parseTraceID(value)
			if err != nil {
				return empty, errMalformedTraceID
			}
		} else if strings.HasPrefix(part, parentIDKey) {
			//extract parentId
			sc.SpanID, err = trace.SpanIDFromHex(value)
			if err != nil {
				return empty, errInvalidSpanIDLength
			}
		} else if strings.HasPrefix(part, sampleFlagKey) {
			//extract traceflag
			sc.TraceFlags = parseTraceFlag(value)
		}
	}
	return sc, nil
}

// returns position of the first occurrence of a substring starting at pos index
func indexOf(str string, substr string, pos int) int {
	index := strings.Index(str[pos:], substr)
	if index > -1 {
		index += pos
	}
	return index
}

// returns trace Id if  valid else return invalid trace Id
func parseTraceID(xrayTraceID string) (trace.ID, error) {
	if len(xrayTraceID) != traceIDLength {
		return empty.TraceID, errMalformedTraceID
	}
	if !strings.HasPrefix(xrayTraceID, traceIDVersion) {
		return empty.TraceID, errMalformedTraceID
	}

	if xrayTraceID[traceIDDelimitterIndex1:traceIDDelimitterIndex1+1] != traceIDDelimiter ||
		xrayTraceID[traceIDDelimitterIndex2:traceIDDelimitterIndex2+1] != traceIDDelimiter {
		return empty.TraceID, errMalformedTraceID
	}

	epochPart := xrayTraceID[traceIDDelimitterIndex1+1 : traceIDDelimitterIndex2]
	uniquePart := xrayTraceID[traceIDDelimitterIndex2+1 : traceIDLength]

	result := epochPart + uniquePart
	return trace.IDFromHex(result)
}

// returns traceFlag
func parseTraceFlag(xraySampledFlag string) byte {
	if len(xraySampledFlag) == sampledFlagLength && xraySampledFlag != isSampled {
		return traceFlagNone
	}
	return trace.FlagsSampled
}

func (awsxray Xray) Fields() []string {
	return []string{traceHeaderKey}
}
