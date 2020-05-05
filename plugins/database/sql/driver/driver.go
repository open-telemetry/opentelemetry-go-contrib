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

package driver

import (
	"database/sql/driver"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"go.opentelemetry.io/contrib/internal/trace"
	otelcore "go.opentelemetry.io/otel/api/core"
	otelkey "go.opentelemetry.io/otel/api/key"
)

// NewDriver returns a tracing driver that utilizes the passed driver
// for the heavy lifting.
func NewDriver(realDriver driver.Driver, opts ...Option) driver.Driver {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	setup := setupFromConfig(&cfg)
	return newDriver(realDriver, setup)
}

// driver.Driver functions for driver.Conn

func traceDDOpen(r driver.Driver, setup *tracingSetup, name string) (driver.Conn, error) {
	attrs := attributesFromName(name)
	connSetup := setup.setupWithExtraAttrs(attrs...)
	ctx, span := connSetup.StartNoCtxNoStmt("open")
	realConn, err := r.Open(name)
	if err != nil {
		// set status error
		span.RecordError(ctx, err)
	}
	return maybeNewConn(realConn, setup), err
}

// driver.Driver functions for driver.DriverContext

func traceDDOpenConnector(r driver.DriverContext, setup *tracingSetup, name string) (driver.Connector, error) {
	realConnector, err := r.OpenConnector(name)
	if err != nil {
		return nil, err
	}
	attrs := attributesFromName(name)
	connSetup := setup.setupWithExtraAttrs(attrs...)
	return newConnector(realConnector, setup, connSetup), nil
}

func attributesFromName(name string) []otelcore.KeyValue {
	if attrs, ok := trySpaceSeparatedOpts(name); ok {
		return attrs
	}
	if attrs, ok := tryURLBasedName(name); ok {
		return attrs
	}
	return nil
}

func trySpaceSeparatedOpts(name string) ([]otelcore.KeyValue, bool) {
	fields := strings.Split(name, " ")
	var attrs []otelcore.KeyValue
	u := url.URL{
		Scheme: "db",
	}
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]
		kv := dbAttrFromPair(key, value, &u)
		if kv.Value.Type() != otelcore.INVALID {
			attrs = append(attrs, kv)
		}
	}
	if len(attrs) == 0 {
		return nil, false
	}
	attrs = append(attrs, otelkey.String("db.url", u.String()))
	return attrs, true
}

func dbAttrFromPair(key, value string, u *url.URL) otelcore.KeyValue {
	switch key {
	case "user":
		u.User = url.User(value)
		return otelkey.String("db.user", value)
	case "database":
		u.Path = fmt.Sprintf("/%s", value)
		return otelkey.String("db.instance", value)
	case "host":
		if u.Host != "" {
			u.Host = fmt.Sprintf("%s:%s", value, u.Host)
		} else {
			u.Host = value
		}
		if ip := net.ParseIP(value); ip != nil {
			return otelkey.String("net.peer.ip", value)
		}
		return otelkey.String("net.peer.name", value)
	case "port":
		numPort, err := strconv.ParseUint(value, 10, 16)
		if err == nil {
			if u.Host != "" {
				u.Host = fmt.Sprintf("%s:%d", u.Host, numPort)
			} else {
				u.Host = value
			}
			return otelkey.Int("net.peer.port", (int)(numPort))
		}
	case "sslmode":
		if u.RawQuery == "" {
			u.RawQuery = fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(value))
		} else {
			u.RawQuery = fmt.Sprintf("%s&%s=%s", u.RawQuery, url.QueryEscape(key), url.QueryEscape(value))
		}
	}
	return otelcore.KeyValue{}
}

func tryURLBasedName(name string) ([]otelcore.KeyValue, bool) {
	u, err := url.Parse(name)
	if err != nil {
		return nil, false
	}
	var attrs []otelcore.KeyValue
	if u.User != nil {
		attrs = append(attrs, otelkey.String("db.user", u.User.Username()))
	}
	if instance := strings.TrimPrefix(u.Path, "/"); instance != "" {
		attrs = append(attrs, otelkey.String("db.instance", instance))
	}
	attrs = append(attrs, trace.NetPeerAttrsFromString(u.Host, trace.HostParsePermissive)...)
	attrs = append(attrs, otelkey.String("db.url", u.String()))
	return attrs, true
}
