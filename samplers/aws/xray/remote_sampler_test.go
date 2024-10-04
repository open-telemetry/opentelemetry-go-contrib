// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xray

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRemoteSamplerDescription assert remote sampling description.
func TestRemoteSamplerDescription(t *testing.T) {
	rs := &remoteSampler{}

	s := rs.Description()
	assert.Equal(t, "AWSXRayRemoteSampler{remote sampling with AWS X-Ray}", s)
}

func TestNewRemoteSamplerDescription(t *testing.T) {
	endpointUrl, _ := url.Parse("http://localhost:2000")
	rs, _ := NewRemoteSampler(context.Background(), "otel-test", "", WithEndpoint(*endpointUrl), WithSamplingRulesPollingInterval(300*time.Second))

	s := rs.Description()
	assert.Equal(t, "ParentBased{root:AWSXRayRemoteSampler{remote sampling with AWS X-Ray},remoteParentSampled:AlwaysOnSampler,remoteParentNotSampled:AlwaysOffSampler,localParentSampled:AlwaysOnSampler,localParentNotSampled:AlwaysOffSampler}", s)
}
