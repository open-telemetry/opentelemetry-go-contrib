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

func (d *testDetector) Detect(context.Context) (*resource.Resource, error) {
	return resource.NewWithAttributes(d.schemaURL, d.attr...), d.err
}

func TestRegisterAndDetector(t *testing.T) {
	id := ID("custom")
	Register(id, testFactory())

	detector, err := Detector(id)
	if err != nil {
		t.Fatalf("got error: %v, expected no error", err)
	}

	c, ok := detector.(*composite)
	if !ok {
		t.Errorf("got %T, expected composite detector", detector)
	}

	if len(c.detectors) != 1 {
		t.Fatalf("got %d detectors, expected 1 detector", len(c.detectors))
	}

	if _, ok := c.detectors[0].(*testDetector); !ok {
		t.Errorf("got %T, expected testDetector", c.detectors[0])
	}
}

func TestRegisterDuplicate(t *testing.T) {
	id := ID("duplicate")
	Register(id, testFactory())

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("got no panic, expected panic for duplicate registration")
		}
	}()
	Register(id, testFactory())
}

func TestParse(t *testing.T) {
	ids := make([]ID, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}

	detector, err := Detector(ids...)
	if err != nil {
		t.Fatalf("got error: %v, expected no error", err)
	}

	c, ok := detector.(*composite)
	if !ok {
		t.Errorf("got %T, expected composite detector", detector)
	}

	if len(c.detectors) != len(registry) {
		t.Errorf("got %d detectors, expected %d detectors", len(c.detectors), len(registry))
	}
}

func TestParseUnknown(t *testing.T) {
	_, err := Detector(ID("unknown"))
	if !errors.Is(err, ErrUnknownDetector) {
		t.Errorf("got %v, expected ErrUnknownDetector", err)
	}
}

func TestOptDetectorDetect(t *testing.T) {
	want := attribute.String("key", "value")
	opt := resource.WithAttributes(want)
	detector := optDetector{opt: opt}

	res, err := detector.Detect(t.Context())
	if err != nil {
		t.Fatalf("got error: %v, expected no error", err)
	}

	if res == nil {
		t.Errorf("got nil resource, expected non-nil resource")
	}

	got := res.Attributes()
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %v, expected %v", got, []attribute.KeyValue{want})
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

	res, err := comp.Detect(t.Context())
	if !errors.Is(err, knownErr) {
		t.Errorf("got error %v, expected %v", err, knownErr)
	}

	if res == nil {
		t.Errorf("got nil resource, expected non-nil resource")
	}

	got := res.Attributes()
	if len(got) != 2 {
		t.Fatalf("got %d attributes, expected 2 attributes", len(got))
	}

	if got[0].Key != a.Key && got[1].Key != a.Key {
		t.Errorf("got %v, expected attribute %s", got, a.Key)
	}
	if got[0].Key != b.Key && got[1].Key != b.Key {
		t.Errorf("got %v, expected attribute %s", got, b.Key)
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

	res, err := comp.Detect(t.Context())
	if !errors.Is(err, resource.ErrSchemaURLConflict) {
		t.Errorf("got error %v, expected %v", err, resource.ErrSchemaURLConflict)
	}

	if res != nil {
		t.Error("got non-nil resource, expected nil resource")
	}
}
