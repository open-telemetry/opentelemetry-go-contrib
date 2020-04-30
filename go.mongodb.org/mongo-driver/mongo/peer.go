// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package mongo

import "go.opentelemetry.io/otel/api/core"

const (
	// PeerHostname records the host name of the peer.
	PeerHostnameKey = core.Key("peer.hostname")
	// PeerPort records the port number of the peer.
	PeerPortKey = core.Key("peer.port")
)

// PeerHostname records the host name of the peer.
func PeerHostname(peerHostname string) core.KeyValue {
	return PeerHostnameKey.String(peerHostname)
}

// PeerPort records the port number of the peer.
func PeerPort(peerport string) core.KeyValue {
	return PeerPortKey.String(peerport)
}

