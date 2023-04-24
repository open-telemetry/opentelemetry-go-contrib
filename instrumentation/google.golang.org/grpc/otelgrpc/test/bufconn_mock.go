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

package test // import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/test"

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc/test/bufconn"
)

const (
	mockIP   = "1.1.1.1"
	mockPort = 1234
)

var (
	mockAddr, _ = net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", mockIP, mockPort))
)

// bufConnMock wraps a bufconn.Lister for the sake of returning a valid address. This is so we can properly test
// returning net.sock.peer.addr and net.sock.peer.port attributes.
type bufConnMock struct {
	listener *bufconn.Listener
}

func newMockBufConn(size int) *bufConnMock {
	return &bufConnMock{
		listener: bufconn.Listen(size),
	}
}

func (b *bufConnMock) Accept() (net.Conn, error) {
	conn, err := b.listener.Accept()
	if err != nil {
		return nil, err
	}

	return &bufConn{
		conn: conn,
	}, nil
}

func (b *bufConnMock) Close() error {
	return b.listener.Close()
}

func (b *bufConnMock) Addr() net.Addr {
	return mockAddr
}

func (b *bufConnMock) Dial() (net.Conn, error) {
	// bufConnect's listener Dial implementation just calls
	// DialContext under the covers so don't wrap the connection in our mock here
	return b.listener.DialContext(context.Background())
}

func (b *bufConnMock) DialContext(ctx context.Context) (net.Conn, error) {
	conn, err := b.listener.DialContext(ctx)
	if err != nil {
		return nil, err
	}

	return &bufConn{
		conn: conn,
	}, nil
}

type bufConn struct {
	conn net.Conn
}

func (b *bufConn) Read(bytes []byte) (n int, err error) {
	return b.conn.Read(bytes)
}

func (b *bufConn) Write(bytes []byte) (n int, err error) {
	return b.conn.Write(bytes)
}

func (b *bufConn) Close() error {
	return b.conn.Close()
}

func (b *bufConn) LocalAddr() net.Addr {
	return mockAddr
}

func (b *bufConn) RemoteAddr() net.Addr {
	return mockAddr
}

func (b *bufConn) SetDeadline(t time.Time) error {
	return b.conn.SetDeadline(t)
}

func (b *bufConn) SetReadDeadline(t time.Time) error {
	return b.conn.SetReadDeadline(t)
}

func (b *bufConn) SetWriteDeadline(t time.Time) error {
	return b.conn.SetWriteDeadline(t)
}
