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
	"net/http"
	"net/http/httptest"
)

// StartMockAgent runs a mock representation of jaeger-agent.
// This function returns a started server.
func StartMockAgent() (*MockAgent, error) {
	samplingManager := newSamplingManager()
	samplingHandler := &samplingHandler{manager: samplingManager}
	samplingServer := httptest.NewServer(samplingHandler)

	agent := &MockAgent{
		samplingMgr: samplingManager,
		samplingSrv: samplingServer,
	}

	return agent, nil
}

// Close stops the serving of traffic.
func (s *MockAgent) Close() {
	s.samplingSrv.Close()
}

// MockAgent is a mock representation of Jaeger Agent.
// It has an HTTP endpoint for sampling strategies.
type MockAgent struct {
	samplingMgr *samplingManager
	samplingSrv *httptest.Server
}

// SamplingServerAddr returns the host:port of HTTP server exposing sampling strategy endpoint.
func (s *MockAgent) SamplingServerAddr() string {
	return s.samplingSrv.Listener.Addr().String()
}

// AddSamplingStrategy registers a sampling strategy for a service.
func (s *MockAgent) AddSamplingStrategy(service string, strategy interface{}) {
	s.samplingMgr.AddSamplingStrategy(service, strategy)
}

type samplingHandler struct {
	manager *samplingManager
}

func (h *samplingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	services := r.URL.Query()["service"]
	if len(services) == 0 {
		http.Error(w, "'service' parameter is empty", http.StatusBadRequest)
		return
	}
	if len(services) > 1 {
		http.Error(w, "'service' parameter must occur only once", http.StatusBadRequest)
		return
	}
	resp, err := h.manager.GetSamplingStrategy(services[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("Error retrieving strategy: %+v", err), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Cannot marshall Thrift to JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write(data); err != nil {
		return
	}
}
