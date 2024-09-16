// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package minsev // import "go.opentelemetry.io/contrib/processors/minsev"

import (
	"sync/atomic"

	"go.opentelemetry.io/otel/log"
)

// Severity represents a log record severity (also known as log level). Smaller
// numerical values correspond to less severe log records (such as debug
// events), larger numerical values correspond to more severe log records (such
// as errors and critical events).
type Severity int

// Severity values defined by OpenTelemetry.
const (
	// A fine-grained debugging log record. Typically disabled in default
	// configurations.
	SeverityTrace1 Severity = -8 // TRACE
	SeverityTrace2 Severity = -7 // TRACE2
	SeverityTrace3 Severity = -6 // TRACE3
	SeverityTrace4 Severity = -5 // TRACE4

	// A debugging log record.
	SeverityDebug1 Severity = -4 // DEBUG
	SeverityDebug2 Severity = -3 // DEBUG2
	SeverityDebug3 Severity = -2 // DEBUG3
	SeverityDebug4 Severity = -1 // DEBUG4

	// An informational log record. Indicates that an event happened.
	SeverityInfo1 Severity = 0 // INFO
	SeverityInfo2 Severity = 1 // INFO2
	SeverityInfo3 Severity = 2 // INFO3
	SeverityInfo4 Severity = 3 // INFO4

	// A warning log record. Not an error but is likely more important than an
	// informational event.
	SeverityWarn1 Severity = 4 // WARN
	SeverityWarn2 Severity = 5 // WARN2
	SeverityWarn3 Severity = 6 // WARN3
	SeverityWarn4 Severity = 7 // WARN4

	// An error log record. Something went wrong.
	SeverityError1 Severity = 8  // ERROR
	SeverityError2 Severity = 9  // ERROR2
	SeverityError3 Severity = 10 // ERROR3
	SeverityError4 Severity = 11 // ERROR4

	// A fatal log record such as application or system crash.
	SeverityFatal1 Severity = 12 // FATAL
	SeverityFatal2 Severity = 13 // FATAL2
	SeverityFatal3 Severity = 14 // FATAL3
	SeverityFatal4 Severity = 15 // FATAL4

	// Convenience definitions for the base severity of each level.
	SeverityTrace = SeverityTrace1
	SeverityDebug = SeverityDebug1
	SeverityInfo  = SeverityInfo1
	SeverityWarn  = SeverityWarn1
	SeverityError = SeverityError1
	SeverityFatal = SeverityFatal1
)

// Severity returns the receiver translated to a [log.Severity].
//
// It implements [Severitier].
func (s Severity) Severity() log.Severity {
	// Unknown defaults to log.SeverityUndefined.
	return translations[s]
}

var translations = map[Severity]log.Severity{
	SeverityTrace1: log.SeverityTrace1,
	SeverityTrace2: log.SeverityTrace2,
	SeverityTrace3: log.SeverityTrace3,
	SeverityTrace4: log.SeverityTrace4,
	SeverityDebug1: log.SeverityDebug1,
	SeverityDebug2: log.SeverityDebug2,
	SeverityDebug3: log.SeverityDebug3,
	SeverityDebug4: log.SeverityDebug4,
	SeverityInfo1:  log.SeverityInfo1,
	SeverityInfo2:  log.SeverityInfo2,
	SeverityInfo3:  log.SeverityInfo3,
	SeverityInfo4:  log.SeverityInfo4,
	SeverityWarn1:  log.SeverityWarn1,
	SeverityWarn2:  log.SeverityWarn2,
	SeverityWarn3:  log.SeverityWarn3,
	SeverityWarn4:  log.SeverityWarn4,
	SeverityError1: log.SeverityError1,
	SeverityError2: log.SeverityError2,
	SeverityError3: log.SeverityError3,
	SeverityError4: log.SeverityError4,
	SeverityFatal1: log.SeverityFatal1,
	SeverityFatal2: log.SeverityFatal2,
	SeverityFatal3: log.SeverityFatal3,
	SeverityFatal4: log.SeverityFatal4,
}

// A SeverityVar is a [Severity] variable, to allow a [LogProcessor] severity
// to change dynamically. It implements [Severitier] as well as a Set method,
// and it is safe for use by multiple goroutines.
//
// The zero SeverityVar corresponds to [SeverityInfo].
type SeverityVar struct {
	val atomic.Int64
}

// Severity returns v's severity.
func (v *SeverityVar) Severity() log.Severity {
	return Severity(int(v.val.Load())).Severity()
}

// Set sets v's Severity to l.
func (v *SeverityVar) Set(l Severity) {
	v.val.Store(int64(l))
}

// A Severitier provides a [log.Severity] value.
type Severitier interface {
	Severity() log.Severity
}
