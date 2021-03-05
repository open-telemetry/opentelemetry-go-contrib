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

package otelmongo

import "go.opentelemetry.io/otel/attribute"

const (
	// PeerHostname records the host name of the peer.
	PeerHostnameKey = attribute.Key("peer.hostname")
	// PeerPort records the port number of the peer.
	PeerPortKey = attribute.Key("peer.port")
)

// PeerHostname records the host name of the peer.
func PeerHostname(peerHostname string) attribute.KeyValue {
	return PeerHostnameKey.String(peerHostname)
}

// PeerPort records the port number of the peer.
func PeerPort(peerport string) attribute.KeyValue {
	return PeerPortKey.String(peerport)
}
