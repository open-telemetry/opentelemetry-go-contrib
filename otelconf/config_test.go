// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestUnmarshalCardinalityLimits(t *testing.T) {
	for _, tt := range []struct {
		name       string
		yamlConfig []byte
		jsonConfig []byte
		wantErr    string
	}{
		{
			name:       "valid with all fields positive",
			jsonConfig: []byte(`{"counter":100,"default":200,"gauge":300,"histogram":400,"observable_counter":500,"observable_gauge":600,"observable_up_down_counter":700,"up_down_counter":800}`),
			yamlConfig: []byte("counter: 100\ndefault: 200\ngauge: 300\nhistogram: 400\nobservable_counter: 500\nobservable_gauge: 600\nobservable_up_down_counter: 700\nup_down_counter: 800"),
		},
		{
			name:       "valid with single field",
			jsonConfig: []byte(`{"default":2000}`),
			yamlConfig: []byte("default: 2000"),
		},
		{
			name:       "valid empty",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("counter: !!str 2000"),
			wantErr:    "unmarshaling error cardinality_limit",
		},
		{
			name:       "invalid counter zero",
			jsonConfig: []byte(`{"counter":0}`),
			yamlConfig: []byte("counter: 0"),
			wantErr:    "field counter: must be > 0",
		},
		{
			name:       "invalid counter negative",
			jsonConfig: []byte(`{"counter":-1}`),
			yamlConfig: []byte("counter: -1"),
			wantErr:    "field counter: must be > 0",
		},
		{
			name:       "invalid default zero",
			jsonConfig: []byte(`{"default":0}`),
			yamlConfig: []byte("default: 0"),
			wantErr:    "field default: must be > 0",
		},
		{
			name:       "invalid default negative",
			jsonConfig: []byte(`{"default":-1}`),
			yamlConfig: []byte("default: -1"),
			wantErr:    "field default: must be > 0",
		},
		{
			name:       "invalid gauge zero",
			jsonConfig: []byte(`{"gauge":0}`),
			yamlConfig: []byte("gauge: 0"),
			wantErr:    "field gauge: must be > 0",
		},
		{
			name:       "invalid gauge negative",
			jsonConfig: []byte(`{"gauge":-1}`),
			yamlConfig: []byte("gauge: -1"),
			wantErr:    "field gauge: must be > 0",
		},
		{
			name:       "invalid histogram zero",
			jsonConfig: []byte(`{"histogram":0}`),
			yamlConfig: []byte("histogram: 0"),
			wantErr:    "field histogram: must be > 0",
		},
		{
			name:       "invalid histogram negative",
			jsonConfig: []byte(`{"histogram":-1}`),
			yamlConfig: []byte("histogram: -1"),
			wantErr:    "field histogram: must be > 0",
		},
		{
			name:       "invalid observable_counter zero",
			jsonConfig: []byte(`{"observable_counter":0}`),
			yamlConfig: []byte("observable_counter: 0"),
			wantErr:    "field observable_counter: must be > 0",
		},
		{
			name:       "invalid observable_counter negative",
			jsonConfig: []byte(`{"observable_counter":-1}`),
			yamlConfig: []byte("observable_counter: -1"),
			wantErr:    "field observable_counter: must be > 0",
		},
		{
			name:       "invalid observable_gauge zero",
			jsonConfig: []byte(`{"observable_gauge":0}`),
			yamlConfig: []byte("observable_gauge: 0"),
			wantErr:    "field observable_gauge: must be > 0",
		},
		{
			name:       "invalid observable_gauge negative",
			jsonConfig: []byte(`{"observable_gauge":-1}`),
			yamlConfig: []byte("observable_gauge: -1"),
			wantErr:    "field observable_gauge: must be > 0",
		},
		{
			name:       "invalid observable_up_down_counter zero",
			jsonConfig: []byte(`{"observable_up_down_counter":0}`),
			yamlConfig: []byte("observable_up_down_counter: 0"),
			wantErr:    "field observable_up_down_counter: must be > 0",
		},
		{
			name:       "invalid observable_up_down_counter negative",
			jsonConfig: []byte(`{"observable_up_down_counter":-1}`),
			yamlConfig: []byte("observable_up_down_counter: -1"),
			wantErr:    "field observable_up_down_counter: must be > 0",
		},
		{
			name:       "invalid up_down_counter zero",
			jsonConfig: []byte(`{"up_down_counter":0}`),
			yamlConfig: []byte("up_down_counter: 0"),
			wantErr:    "field up_down_counter: must be > 0",
		},
		{
			name:       "invalid up_down_counter negative",
			jsonConfig: []byte(`{"up_down_counter":-1}`),
			yamlConfig: []byte("up_down_counter: -1"),
			wantErr:    "field up_down_counter: must be > 0",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := CardinalityLimits{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			cl = CardinalityLimits{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUnmarshalSpanLimits(t *testing.T) {
	for _, tt := range []struct {
		name       string
		yamlConfig []byte
		jsonConfig []byte
		wantErr    string
	}{
		{
			name:       "valid with all fields positive",
			jsonConfig: []byte(`{"attribute_count_limit":100,"attribute_value_length_limit":200,"event_attribute_count_limit":300,"event_count_limit":400,"link_attribute_count_limit":500,"link_count_limit":600}`),
			yamlConfig: []byte("attribute_count_limit: 100\nattribute_value_length_limit: 200\nevent_attribute_count_limit: 300\nevent_count_limit: 400\nlink_attribute_count_limit: 500\nlink_count_limit: 600"),
		},
		{
			name:       "valid with single field",
			jsonConfig: []byte(`{"attribute_value_length_limit":2000}`),
			yamlConfig: []byte("attribute_value_length_limit: 2000"),
		},
		{
			name:       "valid empty",
			jsonConfig: []byte(`{}`),
			yamlConfig: []byte("{}"),
		},
		{
			name:       "invalid data",
			jsonConfig: []byte(`{:2000}`),
			yamlConfig: []byte("attribute_count_limit: !!str 2000"),
			wantErr:    "unmarshaling error span_limit",
		},
		{
			name:       "invalid attribute_count_limit negative",
			jsonConfig: []byte(`{"attribute_count_limit":-1}`),
			yamlConfig: []byte("attribute_count_limit: -1"),
			wantErr:    "field attribute_count_limit: must be >= 0",
		},
		{
			name:       "invalid attribute_value_length_limit negative",
			jsonConfig: []byte(`{"attribute_value_length_limit":-1}`),
			yamlConfig: []byte("attribute_value_length_limit: -1"),
			wantErr:    "field attribute_value_length_limit: must be >= 0",
		},
		{
			name:       "invalid event_attribute_count_limit negative",
			jsonConfig: []byte(`{"event_attribute_count_limit":-1}`),
			yamlConfig: []byte("event_attribute_count_limit: -1"),
			wantErr:    "field event_attribute_count_limit: must be >= 0",
		},
		{
			name:       "invalid event_count_limit negative",
			jsonConfig: []byte(`{"event_count_limit":-1}`),
			yamlConfig: []byte("event_count_limit: -1"),
			wantErr:    "field event_count_limit: must be >= 0",
		},
		{
			name:       "invalid link_attribute_count_limit negative",
			jsonConfig: []byte(`{"link_attribute_count_limit":-1}`),
			yamlConfig: []byte("link_attribute_count_limit: -1"),
			wantErr:    "field link_attribute_count_limit: must be >= 0",
		},
		{
			name:       "invalid link_count_limit negative",
			jsonConfig: []byte(`{"link_count_limit":-1}`),
			yamlConfig: []byte("link_count_limit: -1"),
			wantErr:    "field link_count_limit: must be >= 0",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl := SpanLimits{}
			err := cl.UnmarshalJSON(tt.jsonConfig)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			cl = SpanLimits{}
			err = yaml.Unmarshal(tt.yamlConfig, &cl)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
