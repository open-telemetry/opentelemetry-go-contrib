// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray // import "go.opentelemetry.io/contrib/propagators/aws/xray"
import (
	"context"
	"errors"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	traceHeaderKey       = "X-Amzn-Trace-Id"
	traceHeaderDelimiter = ";"
	kvDelimiter          = "="
	lineageDelimiter     = ":"
	traceIDKey           = "Root"
	sampleFlagKey        = "Sampled"
	parentIDKey          = "Parent"
	lineageKey           = "Lineage"
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

	lineageMaxLength         = 18
	lineageMinLength         = 12
	lineageHashLength        = 8
	lineageMaxLoopCounter    = 32767
	lineageMaxRequestCounter = 255
	lineageMinCounter        = 0
	invalidLineage           = "-1:11111111:0"
)

var (
	empty                    = trace.SpanContext{}
	errInvalidTraceHeader    = errors.New("invalid X-Amzn-Trace-Id header value, should contain 3 different part separated by ;")
	errMalformedTraceID      = errors.New("cannot decode trace ID from header")
	errLengthTraceIDHeader   = errors.New("incorrect length of X-Ray trace ID found, 35 character length expected")
	errInvalidTraceIDVersion = errors.New("invalid X-Ray trace ID header found, does not have valid trace ID version")
	errInvalidSpanIDLength   = errors.New("invalid span ID length, must be 16")
)

// Propagator serializes Span Context to/from AWS X-Ray headers.
//
// Example AWS X-Ray format:
//
// X-Amzn-Trace-Id: Root={traceId};Parent={parentId};Sampled={samplingFlag}.
type Propagator struct{}

// Asserts that the propagator implements the otel.TextMapPropagator interface at compile time.
var _ propagation.TextMapPropagator = &Propagator{}

// Inject injects a context to the carrier following AWS X-Ray format.
func (xray Propagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.TraceID().IsValid() || !sc.SpanID().IsValid() {
		return
	}
	otTraceID := sc.TraceID().String()
	xrayTraceID := traceIDVersion + traceIDDelimiter + otTraceID[0:traceIDFirstPartLength] +
		traceIDDelimiter + otTraceID[traceIDFirstPartLength:]
	parentID := sc.SpanID()
	samplingFlag := notSampled
	if sc.TraceFlags() == traceFlagSampled {
		samplingFlag = isSampled
	}

	headers := []string{
		traceIDKey, kvDelimiter, xrayTraceID, traceHeaderDelimiter, parentIDKey,
		kvDelimiter, parentID.String(), traceHeaderDelimiter, sampleFlagKey, kvDelimiter, samplingFlag,
	}

	contextBaggage := baggage.FromContext(ctx)
	lineage := contextBaggage.Member(lineageKey)

	if lineage.Key() != "" {
		headers = append(headers, traceHeaderDelimiter, lineageKey, kvDelimiter, lineage.Value())
	}

	carrier.Set(traceHeaderKey, strings.Join(headers, "")[:256])
}

// Extract gets a context from the carrier if it contains AWS X-Ray headers.
func (xray Propagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	// extract tracing information
	if header := carrier.Get(traceHeaderKey); header != "" {
		newContext, sc, err := extract(ctx, header)
		if err == nil && sc.IsValid() {
			return trace.ContextWithRemoteSpanContext(newContext, sc)
		}
	}
	return ctx
}

// extract extracts Span Context from context.
func extract(ctx context.Context, headerVal string) (context.Context, trace.SpanContext, error) {
	var (
		scc            = trace.SpanContextConfig{}
		err            error
		delimiterIndex int
		part           string
	)

	currBaggage := baggage.FromContext(ctx)

	pos := 0
	for pos < len(headerVal) {
		delimiterIndex = indexOf(headerVal, traceHeaderDelimiter, pos)
		if delimiterIndex >= 0 {
			part = headerVal[pos:delimiterIndex]
			pos = delimiterIndex + 1
		} else {
			// last part
			part = strings.TrimSpace(headerVal[pos:])
			pos = len(headerVal)
		}
		equalsIndex := strings.Index(part, kvDelimiter)
		if equalsIndex < 0 {
			return ctx, empty, errInvalidTraceHeader
		}
		value := part[equalsIndex+1:]
		if strings.HasPrefix(part, traceIDKey) {
			scc.TraceID, err = parseTraceID(value)
			if err != nil {
				return ctx, empty, err
			}
		} else if strings.HasPrefix(part, parentIDKey) {
			// extract parentId
			scc.SpanID, err = trace.SpanIDFromHex(value)
			if err != nil {
				return ctx, empty, errInvalidSpanIDLength
			}
		} else if strings.HasPrefix(part, sampleFlagKey) {
			// extract traceflag
			scc.TraceFlags = parseTraceFlag(value)
		} else if strings.HasPrefix(part, lineageKey) {
			// extract lineage
			lineageHeader := parseLineageHeader(value)
			if isValidLineage(lineageHeader) {
				lineageBaggage, _ := baggage.NewMember(lineageKey, lineageHeader)
				currBaggage, _ = currBaggage.SetMember(lineageBaggage)
			}
		}
	}

	if currBaggage.Len() > 0 {
		ctx = baggage.ContextWithBaggage(ctx, currBaggage)
	}

	return ctx, trace.NewSpanContext(scc), nil
}

// indexOf returns position of the first occurrence of a substr in str starting at pos index.
func indexOf(str string, substr string, pos int) int {
	index := strings.Index(str[pos:], substr)
	if index > -1 {
		index += pos
	}
	return index
}

// parseTraceID returns trace ID if  valid else return invalid trace ID.
func parseTraceID(xrayTraceID string) (trace.TraceID, error) {
	if len(xrayTraceID) != traceIDLength {
		return empty.TraceID(), errLengthTraceIDHeader
	}
	if !strings.HasPrefix(xrayTraceID, traceIDVersion) {
		return empty.TraceID(), errInvalidTraceIDVersion
	}

	if xrayTraceID[traceIDDelimitterIndex1:traceIDDelimitterIndex1+1] != traceIDDelimiter ||
		xrayTraceID[traceIDDelimitterIndex2:traceIDDelimitterIndex2+1] != traceIDDelimiter {
		return empty.TraceID(), errMalformedTraceID
	}

	epochPart := xrayTraceID[traceIDDelimitterIndex1+1 : traceIDDelimitterIndex2]
	uniquePart := xrayTraceID[traceIDDelimitterIndex2+1 : traceIDLength]

	result := epochPart + uniquePart
	return trace.TraceIDFromHex(result)
}

// parseTraceFlag returns a parsed trace flag.
func parseTraceFlag(xraySampledFlag string) trace.TraceFlags {
	if len(xraySampledFlag) == sampledFlagLength && xraySampledFlag != isSampled {
		return traceFlagNone
	}
	return trace.FlagsSampled
}

func parseLineageHeader(xrayLineageHeader string) string {
	numOfDelimiters := strings.Count(xrayLineageHeader, lineageDelimiter)
	if len(xrayLineageHeader) < lineageMinLength ||
		len(xrayLineageHeader) > lineageMaxLength ||
		numOfDelimiters != 2 {
		return invalidLineage
	}

	return xrayLineageHeader
}

func isValidLineage(lineage string) bool {
	split := strings.Split(lineage, lineageDelimiter)
	hash := split[1]
	loopCounter := parseIntWithBase(split[0], 10)
	requestCounter := parseIntWithBase(split[2], 10)

	isHashValid := len(hash) == lineageHashLength && parseIntWithBase(hash, 16) != -1
	isValidRequestCounter := requestCounter <= lineageMaxRequestCounter && requestCounter >= lineageMinCounter
	isValidLoopCounter := loopCounter <= lineageMaxLoopCounter && loopCounter >= lineageMinCounter

	return isHashValid && isValidRequestCounter && isValidLoopCounter
}

func parseIntWithBase(s string, base int) int64 {
	val, err := strconv.ParseInt(s, base, 64)

	if err != nil {
		return -1
	}

	return val
}

// Fields returns list of fields used by HTTPTextFormat.
func (xray Propagator) Fields() []string {
	return []string{traceHeaderKey}
}
