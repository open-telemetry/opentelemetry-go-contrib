// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

type cloudRunTestCtxKey struct{}

var (
	notOnGCE = func(_ context.Context) bool { return false }
	onGCE    = func(_ context.Context) bool { return true }
)

func getenv(m map[string]string) func(string) string {
	return func(s string) string {
		if m == nil {
			return ""
		}
		return m[s]
	}
}

type client struct {
	m        map[string]string
	contexts []context.Context
}

func setupForTest(c *CloudRun, mc metadataClient, ongce func(context.Context) bool, getenv func(string) string) {
	c.mc = mc
	c.onGCE = ongce
	c.getenv = getenv
}

func (c *client) GetWithContext(ctx context.Context, s string) (string, error) {
	c.contexts = append(c.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	got, ok := c.m[s]
	if !ok {
		return "", fmt.Errorf("%q do not exist", s)
	} else if got == "" {
		return "", fmt.Errorf("%q is empty", s)
	}
	return got, nil
}

func (c *client) InstanceIDWithContext(ctx context.Context) (string, error) {
	return c.GetWithContext(ctx, "instance/id")
}

func (c *client) ProjectIDWithContext(ctx context.Context) (string, error) {
	return c.GetWithContext(ctx, "project/project-id")
}

var _ metadataClient = (*client)(nil)

func TestCloudRunDetectorNotOnGCE(t *testing.T) {
	ctx := t.Context()
	c := NewCloudRun()
	setupForTest(c, nil, notOnGCE, getenv(nil))

	if res, err := c.Detect(ctx); res != nil || err != nil {
		t.Errorf("Expect c.Detect(ctx) to return (nil, nil), got (%v, %v)", res, err)
	}
}

func TestCloudRunDetectorExpectSuccess(t *testing.T) {
	ctx := t.Context()

	metadata := map[string]string{
		"project/project-id": "foo",
		"instance/id":        "bar",
		"instance/region":    "/projects/123/regions/utopia",
	}
	envvars := map[string]string{
		"K_SERVICE": "x-service",
	}
	want, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("cloud.account.id", "foo"),
			attribute.String("cloud.provider", "gcp"),
			attribute.String("cloud.region", "utopia"),
			attribute.String("service.instance.id", "bar"),
			attribute.String("service.name", "x-service"),
			attribute.String("service.namespace", "cloud-run-managed"),
		),
	)
	if err != nil {
		t.Fatalf("failed to create a resource: %v", err)
	}
	c := NewCloudRun()
	setupForTest(c, &client{m: metadata}, onGCE, getenv(envvars))

	if res, err := c.Detect(ctx); err != nil {
		t.Fatalf("got unexpected failure: %v", err)
	} else if diff := cmp.Diff(want, res); diff != "" {
		t.Errorf("detected resource differ from expected (-want, +got)\n%s", diff)
	}
}

func TestCloudRunDetectorExpectFail(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name     string
		metadata map[string]string
		envvars  map[string]string
	}{
		{
			name: "Missing ProjectID",
			metadata: map[string]string{
				"instance/id":     "bar",
				"instance/region": "utopia",
			},
			envvars: map[string]string{
				"K_SERVICE": "x-service",
			},
		},
		{
			name: "Missing InstanceID",
			metadata: map[string]string{
				"project/project-id": "foo",
				"instance/region":    "utopia",
			},
			envvars: map[string]string{
				"K_SERVICE": "x-service",
			},
		},
		{
			name: "Missing Region",
			metadata: map[string]string{
				"project/project-id": "foo",
				"instance/id":        "bar",
			},
			envvars: map[string]string{
				"K_SERVICE": "x-service",
			},
		},
		{
			name: "Missing K_SERVICE envvar",
			metadata: map[string]string{
				"project/project-id": "foo",
				"instance/id":        "bar",
				"instance/region":    "utopia",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := NewCloudRun()
			setupForTest(c, &client{m: test.metadata}, onGCE, getenv(test.envvars))

			if res, err := c.Detect(ctx); err == nil {
				t.Errorf("Expect c.Detect(ctx) to return error, got nil (resource: %v)", res)
			} else {
				t.Logf("err: %v", err)
			}
		})
	}
}

func TestCloudRunDetectorContextPropagation(t *testing.T) {
	metadata := map[string]string{
		"project/project-id": "foo",
		"instance/id":        "bar",
		"instance/region":    "/projects/123/regions/utopia",
	}
	envvars := map[string]string{
		"K_SERVICE": "x-service",
	}

	t.Run("propagates context value to onGCE and metadata getters", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), cloudRunTestCtxKey{}, "cloud-run-ctx")
		var onGCECtx context.Context
		mc := &client{m: metadata}
		c := NewCloudRun()
		setupForTest(c, mc, func(ctx context.Context) bool {
			onGCECtx = ctx
			return true
		}, getenv(envvars))

		if _, err := c.Detect(ctx); err != nil {
			t.Fatalf("got unexpected failure: %v", err)
		}
		if got := onGCECtx.Value(cloudRunTestCtxKey{}); got != "cloud-run-ctx" {
			t.Errorf("onGCE context value = %v, want %q", got, "cloud-run-ctx")
		}
		if len(mc.contexts) != 3 {
			t.Fatalf("expected 3 metadata calls, got %d", len(mc.contexts))
		}
		for i, reqCtx := range mc.contexts {
			if got := reqCtx.Value(cloudRunTestCtxKey{}); got != "cloud-run-ctx" {
				t.Errorf("metadata call %d context value = %v, want %q", i, got, "cloud-run-ctx")
			}
		}
	})

	t.Run("availability probe respects canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		mc := &client{m: metadata}
		c := NewCloudRun()
		setupForTest(c, mc, func(ctx context.Context) bool {
			return ctx.Err() == nil
		}, getenv(envvars))

		if res, err := c.Detect(ctx); res != nil || err != nil {
			t.Errorf("Expect c.Detect(ctx) to return (nil, nil), got (%v, %v)", res, err)
		}
		if len(mc.contexts) != 0 {
			t.Errorf("expected 0 metadata calls when onGCE fails, got %d", len(mc.contexts))
		}
	})

	t.Run("metadata getters report canceled and deadline exceeded errors", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(t.Context())
		cancel()
		expiredCtx, cancelDeadline := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
		defer cancelDeadline()

		for _, tc := range []struct {
			name    string
			ctx     context.Context
			wantErr error
		}{
			{name: "canceled", ctx: canceledCtx, wantErr: context.Canceled},
			{name: "deadline exceeded", ctx: expiredCtx, wantErr: context.DeadlineExceeded},
		} {
			t.Run(tc.name, func(t *testing.T) {
				mc := &client{m: metadata}
				c := NewCloudRun()
				setupForTest(c, mc, onGCE, getenv(envvars))

				_, err := c.Detect(tc.ctx)
				if err == nil || !strings.Contains(err.Error(), tc.wantErr.Error()) {
					t.Errorf("c.Detect(ctx) error = %v, want substring %q", err, tc.wantErr.Error())
				}
				if len(mc.contexts) != 3 {
					t.Errorf("expected 3 metadata calls, got %d", len(mc.contexts))
				}
			})
		}
	})
}
