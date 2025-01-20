package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   *OpenTelemetryConfiguration
		wantErr error
		wantOK  bool
	}{
		{
			name: "empty resource attribute",
			input: &OpenTelemetryConfiguration{
				Resource: &Resource{
					Attributes: []AttributeNameValue{
						{},
					},
				},
			},
			wantErr: errors.New(errCtx + " empty Resource.Attribute[0]"),
		},
		{
			name: "missing resource attribute value",
			input: &OpenTelemetryConfiguration{
				Resource: &Resource{
					Attributes: []AttributeNameValue{
						{
							Name:  "empty value",
							Value: nil,
						},
					},
				},
			},
			wantErr: errors.New(errCtx + " missing Resource.Attribute[0] value"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, ok := validateConfig(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
