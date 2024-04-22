// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectFormat(t *testing.T) {
	detector := New()

	serviceResource, err := detector.Detect(context.Background())
	assert.NoError(t, err)

	var uuid string

	for _, kv := range serviceResource.Attributes() {
		if kv.Key == "service.instance.id" {
			uuid = kv.Value.AsString()
		}
	}

	matched, err := regexp.MatchString("^[A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12}$", uuid)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestDetectRandom(t *testing.T) {
	uuids := map[string] int{}

	for i := 0; i < 10; i++ {
		detector := New()

		serviceResource, err := detector.Detect(context.Background())
		assert.NoError(t, err)

		for _, kv := range serviceResource.Attributes() {
			if kv.Key == "service.instance.id" {
				uuids[kv.Value.AsString()] = 1
			}
		}
	}

	assert.Equal(t, 10, len(uuids))
}
