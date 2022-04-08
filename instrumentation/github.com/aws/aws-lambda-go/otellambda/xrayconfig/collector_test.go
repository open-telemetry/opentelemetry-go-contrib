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

package xrayconfig

// Pared down version of go.opentelemetry.io/otel/exporters/otlp/otlptrace/internal/otlptracetest/collector.go
// for end to end testing

import (
	"sort"

	collectortracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// SpansStorage stores the spans. Mock collectors can use it to
// store spans they have received.
type SpansStorage struct {
	rsm       map[string]*tracepb.ResourceSpans
	spanCount int
}

// NewSpansStorage creates a new spans storage.
func NewSpansStorage() SpansStorage {
	return SpansStorage{
		rsm: make(map[string]*tracepb.ResourceSpans),
	}
}

// AddSpans adds spans to the spans storage.
func (s *SpansStorage) AddSpans(request *collectortracepb.ExportTraceServiceRequest) {
	for _, rs := range request.GetResourceSpans() {
		rstr := resourceString(rs.Resource)
		if existingRs, ok := s.rsm[rstr]; !ok {
			s.rsm[rstr] = rs
			// TODO (rghetia): Add support for library Info.
			if len(rs.ScopeSpans) == 0 {
				rs.ScopeSpans = []*tracepb.ScopeSpans{
					{
						Spans: []*tracepb.Span{},
					},
				}
			}
			s.spanCount += len(rs.ScopeSpans[0].Spans)
		} else {
			if len(rs.ScopeSpans) > 0 {
				newSpans := rs.ScopeSpans[0].GetSpans()
				existingRs.ScopeSpans[0].Spans =
					append(existingRs.ScopeSpans[0].Spans, newSpans...)
				s.spanCount += len(newSpans)
			}
		}
	}
}

// GetSpans returns the stored spans.
func (s *SpansStorage) GetSpans() []*tracepb.Span {
	spans := make([]*tracepb.Span, 0, s.spanCount)
	for _, rs := range s.rsm {
		spans = append(spans, rs.ScopeSpans[0].Spans...)
	}
	return spans
}

// GetResourceSpans returns the stored resource spans.
func (s *SpansStorage) GetResourceSpans() []*tracepb.ResourceSpans {
	rss := make([]*tracepb.ResourceSpans, 0, len(s.rsm))
	for _, rs := range s.rsm {
		rss = append(rss, rs)
	}
	return rss
}

func resourceString(res *resourcepb.Resource) string {
	sAttrs := sortedAttributes(res.GetAttributes())
	rstr := ""
	for _, attr := range sAttrs {
		rstr = rstr + attr.String()
	}
	return rstr
}

func sortedAttributes(attrs []*commonpb.KeyValue) []*commonpb.KeyValue {
	sort.Slice(attrs[:], func(i, j int) bool {
		return attrs[i].Key < attrs[j].Key
	})
	return attrs
}
