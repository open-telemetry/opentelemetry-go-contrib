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
	"fmt"
	"net"
	"sync"
	"time"
)

type reconnectingTCPConn struct {
	ffurl       string
	resolveFunc resolveFunc
	dialFunc    dialFunc

	connMtx   sync.RWMutex
	conn      *net.TCPConn
	destAddr  *net.TCPAddr
	closeChan chan struct{}
}

type resolveFunc func(network string, ffurl string) (*net.TCPAddr, error)
type dialFunc func(network string, laddr, raddr *net.TCPAddr) (*net.TCPConn, error)

func newReconnectingTCPConn(ffurl string, resolveTimeout time.Duration, resolveFunc resolveFunc, dialFunc dialFunc) (*reconnectingTCPConn, error) {
	conn := &reconnectingTCPConn{
		ffurl:       ffurl,
		resolveFunc: resolveFunc,
		dialFunc:    dialFunc,
		closeChan:   make(chan struct{}),
	}
	if err := conn.attemptResolveAndDial(); err != nil {
		fmt.Printf("failed resolving destination address on connection startup, with err: %q. retrying in %s", err.Error(), resolveTimeout)
	}
	go conn.reconnectLoop(resolveTimeout)

	return conn, nil
}

func (c *reconnectingTCPConn) reconnectLoop(resolveTimeout time.Duration) {
	ticker := time.NewTicker(resolveTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeChan:
			return
		case <-ticker.C:
			if err := c.attemptResolveAndDial(); err != nil {
				fmt.Errorf("%s", err.Error())
			}
		}
	}
}

func (c *reconnectingTCPConn) attemptResolveAndDial() error {

	newAddr, err := c.resolveFunc("tcp", c.ffurl)
	if err != nil {
		return fmt.Errorf("failed to resolve new addr for host %q, with err: %w", c.ffurl, err)
	}

	if err := c.attemptDialNewAddr(newAddr); err != nil {
		return fmt.Errorf("failed to dial newly resolved addr '%s', with err: %w", newAddr, err)
	}

	return nil
}

func (c *reconnectingTCPConn) attemptDialNewAddr(newAddr *net.TCPAddr) error {
	connTCP, err := c.dialFunc(newAddr.Network(), nil, newAddr)
	if err != nil {
		return err
	}

	c.connMtx.Lock()
	c.destAddr = newAddr
	prevConn := c.conn
	c.conn = connTCP
	c.connMtx.Unlock()

	if prevConn != nil {
		return prevConn.Close()
	}

	return nil
}

func (c *reconnectingTCPConn) Write(b []byte) (int, error) {
	var bytesWritten int
	var err error

	c.connMtx.RLock()
	conn := c.conn
	c.connMtx.RUnlock()

	if conn == nil {
		// if connection is not initialized indicate this with err in order to hook into retry logic
		err = fmt.Errorf("TCP connection not yet initialized, an address has not been resolved")
	} else {
		// write the data, and if any error is encountered, try to reconnect to fluent service and try again before returning
		bytesWritten, err = conn.Write(b)
	}

	if err == nil {
		return bytesWritten, nil
	}

	if reconnErr := c.attemptResolveAndDial(); reconnErr == nil {
		c.connMtx.RLock()
		conn := c.conn
		c.connMtx.RUnlock()

		return conn.Write(b)
	}

	// return original error if reconn fails
	return bytesWritten, err
}

// Close stops the reconnectLoop and closes the connection
func (c *reconnectingTCPConn) Close() error {
	close(c.closeChan)

	c.connMtx.Lock()
	defer c.connMtx.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}
