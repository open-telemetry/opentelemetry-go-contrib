// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config/v0.3.0"

import (
	"errors"
	"fmt"
	"reflect"
)

func (c *OpenTelemetryConfiguration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias OpenTelemetryConfiguration // Prevents infinite recursion
	aux := &alias{}

	// Decode into alias
	if err := unmarshal(aux); err != nil {
		return err
	}

	// Check for an empty attributes list
	if len(aux.Resource.Attributes) == 0 {
		return fmt.Errorf("error: 'attributes' list cannot be empty")
	}

	// Assign parsed data back to actual struct
	*c = OpenTelemetryConfiguration(*aux)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *AttributeNameValueType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v struct {
		Value interface{}
	}
	if err := unmarshal(&v.Value); err != nil {
		return err
	}
	var ok bool
	for _, expected := range enumValuesAttributeNameValueType {
		if reflect.DeepEqual(v.Value, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("invalid value (expected one of %#v): %#v", enumValuesAttributeNameValueType, v.Value)
	}
	*j = AttributeNameValueType(v)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *AttributeNameValue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias AttributeNameValue
	aux := &alias{}

	if err := unmarshal(aux); err != nil {
		return err
	}

	// Check for empty name
	if aux.Name == "" {
		return fmt.Errorf("error: attribute 'name' cannot be empty")
	}

	*j = AttributeNameValue(*aux)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *NameStringValuePair) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; !ok {
		return errors.New("yaml: cannot unmarshal field name in NameStringValuePair required")
	}
	if _, ok := raw["value"]; !ok {
		return errors.New("yaml: cannot unmarshal field value in NameStringValuePair required")
	}
	var name, value string
	var ok bool
	if name, ok = raw["name"].(string); !ok {
		return errors.New("yaml: cannot unmarshal field name in NameStringValuePair must be string")
	}
	if value, ok = raw["value"].(string); !ok {
		return errors.New("yaml: cannot unmarshal field value in NameStringValuePair must be string")
	}
	*j = NameStringValuePair{
		Name:  name,
		Value: &value,
	}
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *LanguageSpecificInstrumentation) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*j = raw
	return nil
}
