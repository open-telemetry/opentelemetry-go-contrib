// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.
// Copyright 2020 The OpenTelemetry Authors

package mongo

import "go.opentelemetry.io/otel/api/kv"

const (
	// PeerHostname records the host name of the peer.
	PeerHostnameKey = kv.Key("peer.hostname")
	// PeerPort records the port number of the peer.
	PeerPortKey = kv.Key("peer.port")
)

// PeerHostname records the host name of the peer.
func PeerHostname(peerHostname string) kv.KeyValue {
	return PeerHostnameKey.String(peerHostname)
}

// PeerPort records the port number of the peer.
func PeerPort(peerport string) kv.KeyValue {
	return PeerPortKey.String(peerport)
}
