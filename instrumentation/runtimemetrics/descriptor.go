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

package runtimemetrics // import "github.com/open-telemetry/opentelemetry-go-contrib/instrumentation/runtimemetrics"

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

type builtinKind int

var (
	emptySet = attribute.NewSet()

	ErrUnmatchedBuiltin   = fmt.Errorf("builtin unmatched")
	ErrOvermatchedBuiltin = fmt.Errorf("builtin overmatched")
)

const (
	builtinSkip builtinKind = iota
	builtinCounter
	builtinObjectBytesCounter
	builtinUpDownCounter
	builtinGauge
	builtinHistogram
)

type builtinMetricFamily struct {
	pattern string
	matches int
	kind    builtinKind
}

type builtinDescriptor struct {
	families []*builtinMetricFamily
}

func (k builtinKind) String() string {
	switch k {
	case builtinCounter:
		return "counter"
	case builtinObjectBytesCounter:
		return "object/bytes counter"
	case builtinUpDownCounter:
		return "up/down counter"
	case builtinGauge:
		return "gauge"
	case builtinHistogram:
		return "histogram"
	case builtinSkip:
	}
	return "skipped"
}

func toOTelNameAndStatedUnit(nameAndUnit string) (on, un string) {
	on, un, _ = strings.Cut(nameAndUnit, ":")
	return toOTelName(on), un
}

func toOTelName(name string) string {
	return namePrefix + strings.ReplaceAll(name, "/", ".")
}

// attributeName returns "class", "class2", "class3", ...
func attributeName(order int) string {
	if order == 0 {
		return "class"
	}
	return fmt.Sprintf("class%d", order+1)
}

func newBuiltinDescriptor() *builtinDescriptor {
	return &builtinDescriptor{}
}

func (bd *builtinDescriptor) add(pattern string, kind builtinKind) {
	bd.families = append(bd.families, &builtinMetricFamily{
		pattern: pattern,
		kind:    kind,
	})
}

func (bd *builtinDescriptor) singleCounter(pattern string) {
	bd.add(pattern, builtinCounter)
}

func (bd *builtinDescriptor) classesCounter(pattern string) {
	bd.add(pattern, builtinCounter)
}

func (bd *builtinDescriptor) classesUpDownCounter(pattern string) {
	bd.add(pattern, builtinUpDownCounter)
}

func (bd *builtinDescriptor) objectBytesCounter(pattern string) {
	bd.add(pattern, builtinObjectBytesCounter)
}

func (bd *builtinDescriptor) singleUpDownCounter(pattern string) {
	bd.add(pattern, builtinUpDownCounter)
}

func (bd *builtinDescriptor) singleGauge(pattern string) {
	bd.add(pattern, builtinGauge)
}

func (bd *builtinDescriptor) ignorePattern(pattern string) {
	bd.add(pattern, builtinSkip)
}

func (bd *builtinDescriptor) ignoreHistogram(pattern string) {
	bd.add(pattern, builtinHistogram)
}

func (bd *builtinDescriptor) findMatch(goname string) (mname, munit, descPattern string, attrs []attribute.KeyValue, kind builtinKind, _ error) {
	fam, err := bd.findFamily(goname)
	if err != nil {
		return "", "", "", nil, builtinSkip, err
	}
	fam.matches++

	kind = fam.kind

	// Set the name, unit and pattern.
	if wildCnt := strings.Count(fam.pattern, "*"); wildCnt == 0 {
		mname, munit = toOTelNameAndStatedUnit(goname)
		descPattern = goname
	} else if strings.HasSuffix(fam.pattern, ":*") {
		// Special case for bytes/objects w/ same prefix: two
		// counters, different names.  One has "By" (UCUM for
		// "bytes") units and no suffix.  One has no units and
		// a ".objects" suffix.  (In Prometheus, this becomes
		// _objects and _bytes as you would expect.)
		mname, munit = toOTelNameAndStatedUnit(goname)
		descPattern = goname
		kind = builtinCounter
		if munit == "objects" {
			mname += "." + munit
			munit = ""
		}
	} else {
		pfx, sfx, _ := strings.Cut(fam.pattern, "/*:")
		mname = toOTelName(pfx)
		munit = sfx
		asubstr := goname[len(pfx):]
		asubstr = asubstr[1 : len(asubstr)-len(sfx)-1]
		splitVals := strings.Split(asubstr, "/")
		for order, val := range splitVals {
			attrs = append(attrs, attribute.Key(attributeName(order)).String(val))
		}
		// Ignore subtotals
		if splitVals[len(splitVals)-1] == "total" {
			return "", "", "", nil, builtinSkip, nil
		}
		descPattern = fam.pattern
	}

	// Fix the units for UCUM.
	switch munit {
	case "bytes":
		munit = "By"
	case "seconds":
		munit = "s"
	case "":
	default:
		// Pseudo-units
		munit = "{" + munit + "}"
	}

	// Fix the name if it ends in ".classes"
	if strings.HasSuffix(mname, ".classes") {

		s := mname[:len(mname)-len("classes")]

		// Note that ".classes" is (apparently) intended as a generic
		// suffix, while ".cycles" is an exception.
		// The ideal name depends on what we know.
		switch munit {
		case "By":
			// OTel has similar conventions for memory usage, disk usage, etc, so
			// for metrics with /classes/*:bytes we create a .usage metric.
			mname = s + "usage"
		case "{cpu-seconds}":
			// Same argument above, except OTel uses .time for
			// cpu-timing metrics instead of .usage.
			mname = s + "time"
		}
	}

	// Note: we may be returning the special builtinObjectBytes.
	// if it was not fixed for patterns w/ trailing wildcard (see above).
	return mname, munit, descPattern, attrs, kind, err
}

func (bd *builtinDescriptor) findFamily(name string) (family *builtinMetricFamily, _ error) {
	matches := 0

	for _, f := range bd.families {
		pat := f.pattern
		wilds := strings.Count(pat, "*")
		if wilds > 1 {
			return nil, fmt.Errorf("too many wildcards: %s", pat)
		}
		if wilds == 0 && name == pat {
			matches++
			family = f
			continue
		}
		pfx, sfx, _ := strings.Cut(pat, "*")

		if len(name) > len(pat) && strings.HasPrefix(name, pfx) && strings.HasSuffix(name, sfx) {
			matches++
			family = f
			continue
		}
	}
	if matches == 0 {
		return nil, fmt.Errorf("%s: %w", name, ErrUnmatchedBuiltin)
	}
	if matches > 1 {
		return nil, fmt.Errorf("%s: %w", name, ErrOvermatchedBuiltin)
	}
	family.matches++
	return family, nil
}
