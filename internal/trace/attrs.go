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

package trace

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	otelcore "go.opentelemetry.io/otel/api/core"
	otelkey "go.opentelemetry.io/otel/api/key"
)

type HostParseMode int

const (
	HostParseStrict HostParseMode = iota
	HostParsePermissive
)

func NetPeerAttrsFromString(host string, mode HostParseMode) []otelcore.KeyValue {
	return attrsFromString("peer", host, mode)
}

func NetHostAttrsFromString(host string, mode HostParseMode) []otelcore.KeyValue {
	return attrsFromString("host", host, mode)
}

func attrsFromString(infix, host string, mode HostParseMode) []otelcore.KeyValue {
	name, ip, port := "", "", 0
	hostPart := host
	portPart := ""
	if idx := strings.LastIndex(hostPart, ":"); idx >= 0 {
		hostPart = host[:idx]
		portPart = host[idx+1:]
	}
	if hostPart != "" {
		if parsed := net.ParseIP(hostPart); parsed != nil {
			ip = parsed.String()
		} else {
			name = hostPart
		}
		if portPart != "" {
			numPort, err := strconv.ParseUint(portPart, 10, 16)
			if err == nil {
				port = (int)(numPort)
			} else if mode == HostParseStrict {
				name, ip = "", ""
			}
		}
	}
	var attrs []otelcore.KeyValue
	if name != "" {
		attrs = append(attrs, otelkey.String(fmt.Sprintf("net.%s.name", infix), name))
	}
	if ip != "" {
		attrs = append(attrs, otelkey.String(fmt.Sprintf("net.%s.ip", infix), ip))
	}
	if port != 0 {
		attrs = append(attrs, otelkey.Int(fmt.Sprintf("net.%s.port", infix), port))
	}
	return attrs
}
