// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config/v0.3.0"

import (
	"errors"
	"fmt"
	"reflect"
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
	type Plain NameStringValuePair
	var plain Plain
	if err := unmarshal(&plain); err != nil {
		return err
	}
	*j = NameStringValuePair(plain)
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
