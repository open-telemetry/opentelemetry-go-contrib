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

package fluentforward

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockResolver struct {
	mock.Mock
}

func (m *mockResolver) ResolveTCPAddr(network, url string) (*net.TCPAddr, error) {
	args := m.Called(network, url)

	a0 := args.Get(0)
	if a0 == nil {
		return (*net.TCPAddr)(nil), args.Error(1)
	}
	return a0.(*net.TCPAddr), args.Error(1)
}

type mockDialer struct {
	mock.Mock
}

func (m *mockDialer) DialTCP(network string, laddr, raddr *net.TCPAddr) (*net.TCPConn, error) {
	args := m.Called(network, laddr, raddr)

	a0 := args.Get(0)
	if a0 == nil {
		return (*net.TCPConn)(nil), args.Error(1)
	}

	return a0.(*net.TCPConn), args.Error(1)
}

func newTCPConn() (*net.TCPConn, error) {
	addr, err := net.ResolveTCPAddr("tcp", url)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func TestNewResolvedTCPConn(t *testing.T) {

	// url := "localhost:24224"

	mockServer := startMockFluentServer(t)
	defer mockServer.Close()

	clientConn, err := newTCPConn()
	require.NoError(t, err)

	mockTCPAddr := &net.TCPAddr{
		IP:   net.IPv4(1, 2, 3, 4),
		Port: 24224,
	}

	resolver := mockResolver{}
	resolver.
		On("ResolveTCPAddr", "tcp", url).
		Return(mockTCPAddr, nil).
		Once()

	dialer := mockDialer{}
	dialer.
		On("DialTCP", "tcp", (*net.TCPAddr)(nil), mockTCPAddr).
		Return(clientConn, nil).
		Once()

	conn, err := newReconnectingTCPConn(url, time.Hour, resolver.ResolveTCPAddr, dialer.DialTCP)
	assert.NoError(t, err)
	require.NotNil(t, conn)

	err = conn.Close()
	assert.NoError(t, err)

	// assert the actual connection was closed
	assert.Error(t, clientConn.Close())

	resolver.AssertExpectations(t)
	dialer.AssertExpectations(t)
}
