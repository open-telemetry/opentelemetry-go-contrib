// Code created by gotmpl. DO NOT MODIFY.
// source: internal/shared/semconvutil/netconv.go.tmpl

// Copyright The OpenTelemetry Authors
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

package semconvutil // import "go.opentelemetry.io/contrib/instrumentation/gopkg.in/macaron.v1/otelmacaron/internal/semconvutil"

import (
	"net"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// NetTransport returns a trace attribute describing the transport protocol of the
// passed network. See the net.Dial for information about acceptable network
// values.
func NetTransport(network string) attribute.KeyValue {
	return nc.Transport(network)
}

// NetClient returns trace attributes for a client network connection to address.
// See net.Dial for information about acceptable address values, address should
// be the same as the one used to create conn. If conn is nil, only network
// peer attributes will be returned that describe address. Otherwise, the
// socket level information about conn will also be included.
func NetClient(address string, conn net.Conn) []attribute.KeyValue {
	return nc.Client(address, conn)
}

// NetServer returns trace attributes for a network listener listening at address.
// See net.Listen for information about acceptable address values, address
// should be the same as the one used to create ln. If ln is nil, only network
// host attributes will be returned that describe address. Otherwise, the
// socket level information about ln will also be included.
func NetServer(address string, ln net.Listener) []attribute.KeyValue {
	return nc.Server(address, ln)
}

const (
	// ServerAddressKey is the attribute Key conforming to the "server.address"
	// semantic conventions. It represents the logical server hostname, matches
	// server FQDN if available, and IP or socket address if FQDN is not known.
	//
	// Type: string
	// RequirementLevel: Optional
	// Stability: stable
	// Examples: 'example.com'.
	ServerAddressKey = attribute.Key("server.address")

	// ServerPortKey is the attribute Key conforming to the "server.port"
	// semantic conventions. It represents the logical server port number
	//
	// Type: int
	// RequirementLevel: Optional
	// Stability: stable
	// Examples: 80, 8080, 443.
	ServerPortKey = attribute.Key("server.port")

	// ServerSocketDomainKey is the attribute Key conforming to the
	// "server.socket.domain" semantic conventions. It represents the domain
	// name of an immediate peer.
	//
	// Type: string
	// RequirementLevel: Recommended (If different than `server.address`.)
	// Stability: stable
	// Examples: 'proxy.example.com'
	// Note: Typically observed from the client side, and represents a proxy or
	// other intermediary domain name.
	ServerSocketDomainKey = attribute.Key("server.socket.domain")

	// ServerSocketAddressKey is the attribute Key conforming to the
	// "server.socket.address" semantic conventions. It represents the physical
	// server IP address or Unix socket address. If set from the client, should
	// simply use the socket's peer address, and not attempt to find any actual
	// server IP (i.e., if set from client, this may represent some proxy
	// server instead of the logical server).
	//
	// Type: string
	// RequirementLevel: Recommended (If different than `server.address`.)
	// Stability: stable
	// Examples: '10.5.3.2'.
	ServerSocketAddressKey = attribute.Key("server.socket.address")

	// ServerSocketPortKey is the attribute Key conforming to the
	// "server.socket.port" semantic conventions. It represents the physical
	// server port.
	//
	// Type: int
	// RequirementLevel: Recommended (If different than `server.port`.)
	// Stability: stable
	// Examples: 16456.
	ServerSocketPortKey = attribute.Key("server.socket.port")
)

const (
	// ClientSocketAddressKey is the attribute Key conforming to the
	// "client.socket.address" semantic conventions. It represents the
	// immediate client peer address - unix domain socket name, IPv4 or IPv6
	// address.
	//
	// Type: string
	// RequirementLevel: Recommended (If different than `client.address`.)
	// Stability: stable
	// Examples: '/tmp/my.sock', '127.0.0.1'.
	ClientSocketAddressKey = attribute.Key("client.socket.address")

	// ClientSocketPortKey is the attribute Key conforming to the
	// "client.socket.port" semantic conventions. It represents the immediate
	// client peer port number
	//
	// Type: int
	// RequirementLevel: Recommended (If different than `client.port`.)
	// Stability: stable
	// Examples: 35555.
	ClientSocketPortKey = attribute.Key("client.socket.port")
)

// netConv are the network semantic convention attributes defined for a version
// of the OpenTelemetry specification.
type netConv struct {
	NetHostNameKey         attribute.Key
	NetHostPortKey         attribute.Key
	NetPeerNameKey         attribute.Key
	NetPeerPortKey         attribute.Key
	NetSockFamilyKey       attribute.Key
	NetSockPeerAddrKey     attribute.Key
	NetSockPeerPortKey     attribute.Key
	NetSockHostAddrKey     attribute.Key
	NetSockHostPortKey     attribute.Key
	ServerAddressKey       attribute.Key
	ServerPortKey          attribute.Key
	ServerSocketAddressKey attribute.Key
	ServerSocketPortKey    attribute.Key
	ClientSocketAddressKey attribute.Key
	ClientSocketPortKey    attribute.Key
	NetTransportOther      attribute.KeyValue
	NetTransportTCP        attribute.KeyValue
	NetTransportUDP        attribute.KeyValue
	NetTransportInProc     attribute.KeyValue
}

var nc = &netConv{
	NetHostNameKey:         semconv.NetHostNameKey,
	NetHostPortKey:         semconv.NetHostPortKey,
	NetPeerNameKey:         semconv.NetPeerNameKey,
	NetPeerPortKey:         semconv.NetPeerPortKey,
	NetSockFamilyKey:       semconv.NetSockFamilyKey,
	NetSockPeerAddrKey:     semconv.NetSockPeerAddrKey,
	NetSockPeerPortKey:     semconv.NetSockPeerPortKey,
	NetSockHostAddrKey:     semconv.NetSockHostAddrKey,
	NetSockHostPortKey:     semconv.NetSockHostPortKey,
	ServerAddressKey:       ServerAddressKey,
	ServerPortKey:          ServerPortKey,
	ServerSocketAddressKey: ServerSocketAddressKey,
	ServerSocketPortKey:    ServerSocketPortKey,
	ClientSocketAddressKey: ClientSocketAddressKey,
	ClientSocketPortKey:    ClientSocketPortKey,
	NetTransportOther:      semconv.NetTransportOther,
	NetTransportTCP:        semconv.NetTransportTCP,
	NetTransportUDP:        semconv.NetTransportUDP,
	NetTransportInProc:     semconv.NetTransportInProc,
}

func (c *netConv) Transport(network string) attribute.KeyValue {
	switch network {
	case "tcp", "tcp4", "tcp6":
		return c.NetTransportTCP
	case "udp", "udp4", "udp6":
		return c.NetTransportUDP
	case "unix", "unixgram", "unixpacket":
		return c.NetTransportInProc
	default:
		// "ip:*", "ip4:*", and "ip6:*" all are considered other.
		return c.NetTransportOther
	}
}

// Host returns attributes for a network host address.
func (c *netConv) Host(address string) []attribute.KeyValue {
	h, p := splitHostPort(address)
	var n int
	if h != "" {
		n++
		if p > 0 {
			n++
		}
	}

	if n == 0 {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, n)
	attrs = append(attrs, c.HostName(h))
	if p > 0 {
		attrs = append(attrs, c.HostPort(int(p)))
	}
	return attrs
}

// Server returns attributes for a network listener listening at address. See
// net.Listen for information about acceptable address values, address should
// be the same as the one used to create ln. If ln is nil, only network host
// attributes will be returned that describe address. Otherwise, the socket
// level information about ln will also be included.
func (c *netConv) Server(address string, ln net.Listener) []attribute.KeyValue {
	if ln == nil {
		return c.Host(address)
	}

	lAddr := ln.Addr()
	if lAddr == nil {
		return c.Host(address)
	}

	hostName, hostPort := splitHostPort(address)
	sockHostAddr, sockHostPort := splitHostPort(lAddr.String())
	network := lAddr.Network()
	sockFamily := family(network, sockHostAddr)

	n := nonZeroStr(hostName, network, sockHostAddr, sockFamily)
	n += positiveInt(hostPort, sockHostPort)
	attr := make([]attribute.KeyValue, 0, n)
	if hostName != "" {
		attr = append(attr, c.HostName(hostName))
		if hostPort > 0 {
			// Only if net.host.name is set should net.host.port be.
			attr = append(attr, c.HostPort(hostPort))
		}
	}
	if network != "" {
		attr = append(attr, c.Transport(network))
	}
	if sockFamily != "" {
		attr = append(attr, c.NetSockFamilyKey.String(sockFamily))
	}
	if sockHostAddr != "" {
		attr = append(attr, c.NetSockHostAddrKey.String(sockHostAddr))
		if sockHostPort > 0 {
			// Only if net.sock.host.addr is set should net.sock.host.port be.
			attr = append(attr, c.NetSockHostPortKey.Int(sockHostPort))
		}
	}
	return attr
}

func (c *netConv) HostName(name string) attribute.KeyValue {
	return c.NetHostNameKey.String(name)
}

func (c *netConv) HostPort(port int) attribute.KeyValue {
	return c.NetHostPortKey.Int(port)
}

func (c *netConv) ServerAddress(name string) attribute.KeyValue {
	return c.ServerAddressKey.String(name)
}

func (c *netConv) ServerPort(port int) attribute.KeyValue {
	return c.ServerPortKey.Int(port)
}

func (c *netConv) ServerSocketAddress(name string) attribute.KeyValue {
	return c.ServerSocketAddressKey.String(name)
}

func (c *netConv) ServerSocketPort(port int) attribute.KeyValue {
	return c.ServerSocketPortKey.Int(port)
}

func (c *netConv) ClientSocketAddress(name string) attribute.KeyValue {
	return c.ClientSocketAddressKey.String(name)
}

func (c *netConv) ClientSocketPort(port int) attribute.KeyValue {
	return c.ClientSocketPortKey.Int(port)
}

// Client returns attributes for a client network connection to address. See
// net.Dial for information about acceptable address values, address should be
// the same as the one used to create conn. If conn is nil, only network peer
// attributes will be returned that describe address. Otherwise, the socket
// level information about conn will also be included.
func (c *netConv) Client(address string, conn net.Conn) []attribute.KeyValue {
	if conn == nil {
		return c.Peer(address)
	}

	lAddr, rAddr := conn.LocalAddr(), conn.RemoteAddr()

	var network string
	switch {
	case lAddr != nil:
		network = lAddr.Network()
	case rAddr != nil:
		network = rAddr.Network()
	default:
		return c.Peer(address)
	}

	peerName, peerPort := splitHostPort(address)
	var (
		sockFamily   string
		sockPeerAddr string
		sockPeerPort int
		sockHostAddr string
		sockHostPort int
	)

	if lAddr != nil {
		sockHostAddr, sockHostPort = splitHostPort(lAddr.String())
	}

	if rAddr != nil {
		sockPeerAddr, sockPeerPort = splitHostPort(rAddr.String())
	}

	switch {
	case sockHostAddr != "":
		sockFamily = family(network, sockHostAddr)
	case sockPeerAddr != "":
		sockFamily = family(network, sockPeerAddr)
	}

	n := nonZeroStr(peerName, network, sockPeerAddr, sockHostAddr, sockFamily)
	n += positiveInt(peerPort, sockPeerPort, sockHostPort)
	attr := make([]attribute.KeyValue, 0, n)
	if peerName != "" {
		attr = append(attr, c.ServerAddress(peerName))
		if peerPort > 0 {
			// Only if net.peer.name is set should net.peer.port be.
			attr = append(attr, c.ServerPort(peerPort))
		}
	}
	if network != "" {
		attr = append(attr, c.Transport(network))
	}
	if sockFamily != "" {
		attr = append(attr, c.NetSockFamilyKey.String(sockFamily))
	}
	if sockPeerAddr != "" {
		attr = append(attr, c.ServerSocketAddress(sockPeerAddr))
		if sockPeerPort > 0 {
			// Only if net.sock.peer.addr is set should net.sock.peer.port be.
			attr = append(attr, c.ServerSocketPort(sockPeerPort))
		}
	}
	if sockHostAddr != "" {
		attr = append(attr, c.NetSockHostAddrKey.String(sockHostAddr))
		if sockHostPort > 0 {
			// Only if net.sock.host.addr is set should net.sock.host.port be.
			attr = append(attr, c.NetSockHostPortKey.Int(sockHostPort))
		}
	}
	return attr
}

func family(network, address string) string {
	switch network {
	case "unix", "unixgram", "unixpacket":
		return "unix"
	default:
		if ip := net.ParseIP(address); ip != nil {
			if ip.To4() == nil {
				return "inet6"
			}
			return "inet"
		}
	}
	return ""
}

func nonZeroStr(strs ...string) int {
	var n int
	for _, str := range strs {
		if str != "" {
			n++
		}
	}
	return n
}

func positiveInt(ints ...int) int {
	var n int
	for _, i := range ints {
		if i > 0 {
			n++
		}
	}
	return n
}

// Peer returns attributes for a network peer address.
func (c *netConv) Peer(address string) []attribute.KeyValue {
	h, p := splitHostPort(address)
	var n int
	if h != "" {
		n++
		if p > 0 {
			n++
		}
	}

	if n == 0 {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, n)
	attrs = append(attrs, c.PeerName(h))
	if p > 0 {
		attrs = append(attrs, c.PeerPort(int(p)))
	}
	return attrs
}

func (c *netConv) PeerName(name string) attribute.KeyValue {
	return c.NetPeerNameKey.String(name)
}

func (c *netConv) PeerPort(port int) attribute.KeyValue {
	return c.NetPeerPortKey.Int(port)
}

func (c *netConv) SockPeerAddr(addr string) attribute.KeyValue {
	return c.NetSockPeerAddrKey.String(addr)
}

func (c *netConv) SockPeerPort(port int) attribute.KeyValue {
	return c.NetSockPeerPortKey.Int(port)
}

// splitHostPort splits a network address hostport of the form "host",
// "host%zone", "[host]", "[host%zone], "host:port", "host%zone:port",
// "[host]:port", "[host%zone]:port", or ":port" into host or host%zone and
// port.
//
// An empty host is returned if it is not provided or unparsable. A negative
// port is returned if it is not provided or unparsable.
func splitHostPort(hostport string) (host string, port int) {
	port = -1

	if strings.HasPrefix(hostport, "[") {
		addrEnd := strings.LastIndex(hostport, "]")
		if addrEnd < 0 {
			// Invalid hostport.
			return
		}
		if i := strings.LastIndex(hostport[addrEnd:], ":"); i < 0 {
			host = hostport[1:addrEnd]
			return
		}
	} else {
		if i := strings.LastIndex(hostport, ":"); i < 0 {
			host = hostport
			return
		}
	}

	host, pStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return
	}

	p, err := strconv.ParseUint(pStr, 10, 16)
	if err != nil {
		return
	}
	return host, int(p)
}
