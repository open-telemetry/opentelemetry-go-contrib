// Copyright The OpenTelemetry Authors
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

package semconv

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
)

type testServerReq struct {
	hostname   string
	serverPort int
	peerAddr   string
	peerPort   int
	clientIP   string
}

func testTraceRequest(t *testing.T, serv HTTPServer, want func(testServerReq) []attribute.KeyValue) {
	t.Helper()

	got := make(chan *http.Request, 1)
	handler := func(w http.ResponseWriter, r *http.Request) {
		got <- r
		w.WriteHeader(http.StatusOK)
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	srvURL, err := url.Parse(srv.URL)
	require.NoError(t, err)
	srvPort, err := strconv.ParseInt(srvURL.Port(), 10, 32)
	require.NoError(t, err)

	resp, err := srv.Client().Get(srv.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	req := <-got
	peer, peerPort := splitHostPort(req.RemoteAddr)

	const user = "alice"
	req.SetBasicAuth(user, "pswrd")

	const clientIP = "127.0.0.5"
	req.Header.Add("X-Forwarded-For", clientIP)

	srvReq := testServerReq{
		hostname:   srvURL.Hostname(),
		serverPort: int(srvPort),
		peerAddr:   peer,
		peerPort:   peerPort,
		clientIP:   clientIP,
	}

	assert.ElementsMatch(t, want(srvReq), serv.RequestTraceAttrs("", req))
}

func testTraceResponse(t *testing.T, serv HTTPServer, want []attribute.KeyValue) {
	t.Helper()
	emptyResp := ResponseTelemetry{}
	assert.Len(t, serv.ResponseTraceAttrs(emptyResp), 0)

	resp := ResponseTelemetry{
		StatusCode: 200,
		ReadBytes:  701,
		ReadError:  fmt.Errorf("read error"),
		WriteBytes: 802,
		WriteError: fmt.Errorf("write error"),
	}
	assert.ElementsMatch(t, want, serv.ResponseTraceAttrs(resp))
}
