// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright (c) 2021 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils // import "go.opentelemetry.io/contrib/samplers/jaegerremote/internal/testutils"

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// getJSON makes an HTTP call to the specified URL and parses the returned JSON into `out`.
func getJSON(url string, out any) error {
	resp, err := http.Get(url) //nolint:gosec // False positive G107: Potential HTTP request made with variable url
	if err != nil {
		return err
	}
	return readJSON(resp, out)
}

// readJSON reads JSON from http.Response and parses it into `out`.
func readJSON(resp *http.Response, out any) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("status code: %d, body: %s", resp.StatusCode, body)
	}

	if out == nil {
		_, err := io.Copy(io.Discard, resp.Body)
		if err != nil {
			return err
		}
		return nil
	}

	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(out)
}
