package config // import "go.opentelemetry.io/contrib/config/v0.3.0"

import (
	"errors"
	"fmt"
)

const (
	errCtx string = "invalid OpenTelemetryConfiguration:"
)

func validateConfig(config *OpenTelemetryConfiguration) (error, bool) {
	if config == nil {
		return errors.New("invalid OpenTelemetryConfiguration: nil config"), false
	}
	// error on non-empty null values
	if config.Resource != nil {
		for n, attr := range config.Resource.Attributes {
			if attr == (AttributeNameValue{}) {
				return fmt.Errorf("%s empty Resource.Attribute[%d]", errCtx, n), false
			}
			if attr.Value == nil {
				return fmt.Errorf("%s missing Resource.Attribute[%d] value", errCtx, n), false
			}
		}
	}
	return nil, true
}
