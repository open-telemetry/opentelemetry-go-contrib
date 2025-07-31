// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package autodetect_test

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/detectors/autodetect"
)

func init() {
	id := autodetect.ID("my.cfg.detector")
	autodetect.Register(id, func() resource.Detector {
		return MyDetector{}
	})
}

var data = []byte(`{
	"detectors": [
		"host",
		"telemetry.sdk",
		"my.cfg.detector"
	]
}`)

type Config struct {
	Detectors []autodetect.ID `json:"detectors"`
}

func ExampleDetector() {
	// This example shows how to parse resource.Detectors from a user defined
	// configuration file.

	cfg := Config{}
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		panic(err)
	}

	detector, err := autodetect.Detector(cfg.Detectors...)
	if err != nil {
		panic(err)
	}

	// Use the detector as needed.

	res, err := detector.Detect(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Print(enc.Encode(res.Iter()))
	// Output:
	//   host.name my.key telemetry.sdk.language telemetry.sdk.name telemetry.sdk.version
}
