// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autodetect

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

type testDetector struct {
	schemaURL string
	attr      []attribute.KeyValue
	err       error
}

var _ resource.Detector = (*testDetector)(nil)

func testFactory() func() resource.Detector {
	return func() resource.Detector { return &testDetector{} }
}

func (d *testDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	return resource.NewWithAttributes(d.schemaURL, d.attr...), d.err
}

func TestRegisterAndDetector(t *testing.T) {
	id := ID("custom")
	Register(id, testFactory())

	detector, err := Detector(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, ok := detector.(*composite)
	if !ok {
		t.Errorf("expected composite detector, got %T", detector)
	}

	if len(c.detectors) != 1 {
		t.Fatalf("expected 1 detector, got %d", len(c.detectors))
	}

	switch c.detectors[0].(type) {
	case *testDetector:
	default:
		t.Errorf("expected testDetector, got %T", c.detectors[0])
	}
}

func TestRegisterDuplicate(t *testing.T) {
	id := ID("duplicate")
	Register(id, testFactory())

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for duplicate registration")
		}
	}()
	Register(id, testFactory())
}

func TestParse(t *testing.T) {
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, string(id))
	}

	detector, err := Parse(ids...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c, ok := detector.(*composite)
	if !ok {
		t.Errorf("expected composite detector, got %T", detector)
	}

	if len(c.detectors) != len(registry) {
		t.Errorf("expected %d detectors, got %d", len(registry), len(c.detectors))
	}
}

func TestParseUnknown(t *testing.T) {
	_, err := Parse("unknown")
	if !errors.Is(err, ErrUnknownDetector) {
		t.Errorf("expected ErrUnknownDetector, got %v", err)
	}
}

func TestOptDetectorDetect(t *testing.T) {
	want := attribute.String("key", "value")
	opt := resource.WithAttributes(want)
	detector := optDetector{opt: opt}

	res, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil {
		t.Errorf("expected non-nil resource")
	}

	got := res.Attributes()
	if len(got) != 1 || got[0] != want {
		t.Errorf("expected %v, got %v", []attribute.KeyValue{want}, got)
	}
}

func TestCompositeDetect(t *testing.T) {
	a, b := attribute.Int("a", 0), attribute.Int("b", 0)
	knownErr := errors.New("known error")
	detectors := []resource.Detector{
		&testDetector{attr: []attribute.KeyValue{a}},
		&testDetector{
			attr: []attribute.KeyValue{b},
			err:  knownErr,
		},
	}
	comp := newComposite(detectors)

	res, err := comp.Detect(context.Background())
	if !errors.Is(err, knownErr) {
		t.Errorf("expected error %v, got %v", knownErr, err)
	}

	if res == nil {
		t.Errorf("expected non-nil resource")
	}

	got := res.Attributes()
	if len(got) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(got))
	}

	if got[0].Key != a.Key && got[1].Key != a.Key {
		t.Errorf("expected attribute %s, got %v", a.Key, got)
	}
	if got[0].Key != b.Key && got[1].Key != b.Key {
		t.Errorf("expected attribute %s, got %v", b.Key, got)
	}
}

func TestCompositeDetectMergeError(t *testing.T) {
	a, b := attribute.Int("a", 0), attribute.Int("b", 0)
	detectors := []resource.Detector{
		&testDetector{
			schemaURL: "a",
			attr:      []attribute.KeyValue{a},
		},
		&testDetector{
			schemaURL: "b",
			attr:      []attribute.KeyValue{b},
		},
	}
	comp := newComposite(detectors)

	res, err := comp.Detect(context.Background())
	if !errors.Is(err, resource.ErrSchemaURLConflict) {
		t.Errorf("expected error %v, got %v", resource.ErrSchemaURLConflict, err)
	}

	if res != nil {
		t.Error("expected nil resource")
	}
}
