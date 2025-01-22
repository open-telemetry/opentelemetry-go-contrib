// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config/v0.3.0"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

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
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}
	var name string
	var value interface{}
	var ok bool
	if _, ok = raw["name"]; raw != nil && !ok {
		return errors.New("field name in AttributeNameValue: required")
	}
	if name, ok = raw["name"].(string); !ok {
		return errors.New("yaml: cannot unmarshal field name in AttributeNameValue must be string")
	}
	if strings.TrimSpace(name) == "" {
		return errors.New("field name in AttributeNameValue: required")
	}
	if value, ok = raw["value"]; raw != nil && !ok {
		return errors.New("field value in AttributeNameValue: required")
	}
	type Plain AttributeNameValue
	plain := Plain{
		Name:  name,
		Value: value,
	}
	if plain.Type != nil && plain.Type.Value == "int" {
		val, ok := plain.Value.(float64)
		if ok {
			plain.Value = int(val)
		}
	}
	if plain.Type != nil && plain.Type.Value == "int_array" {
		m, ok := plain.Value.([]interface{})
		if ok {
			var vals []interface{}
			for _, v := range m {
				val, ok := v.(float64)
				if ok {
					vals = append(vals, int(val))
				} else {
					vals = append(vals, val)
				}
			}
			plain.Value = vals
		}
	}

	*j = AttributeNameValue(plain)
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
