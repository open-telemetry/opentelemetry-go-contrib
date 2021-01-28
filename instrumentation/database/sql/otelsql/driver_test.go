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

package otelsql

import (
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDriver struct {
	shouldError bool

	openCount, openConnectorCount int
	openConnectorName             string
	openName                      string
}

func newMockDriver(shouldError bool) *mockDriver {
	return &mockDriver{shouldError: shouldError}
}

func (m *mockDriver) OpenConnector(name string) (driver.Connector, error) {
	m.openConnectorName = name
	m.openConnectorCount++
	if m.shouldError {
		return nil, errors.New("openConnector")
	}
	return newMockConnector(m, false), nil
}

func (m *mockDriver) Open(name string) (driver.Conn, error) {
	m.openName = name
	m.openCount++
	if m.shouldError {
		return nil, errors.New("open")
	}
	return newMockConn(false), nil
}

var (
	_ driver.Driver        = (*mockDriver)(nil)
	_ driver.DriverContext = (*mockDriver)(nil)
)

func TestNewDriver(t *testing.T) {
	d := newDriver(newMockDriver(false), config{DBSystem: "test"})

	otelDriver, ok := d.(*otDriver)
	require.True(t, ok)
	assert.Equal(t, newMockDriver(false), otelDriver.driver)
	assert.Equal(t, config{DBSystem: "test"}, otelDriver.cfg)
}

func TestOtDriver_Open(t *testing.T) {
	testCases := []struct {
		name  string
		error bool
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			md := newMockDriver(tc.error)
			d := newDriver(md, config{})

			conn, err := d.Open("test")

			assert.Equal(t, "test", md.openName)
			assert.Equal(t, 1, md.openCount)

			if tc.error {
				assert.Error(t, err)
			} else {
				otelConn, ok := conn.(*otConn)
				require.True(t, ok)
				assert.IsType(t, &mockConn{}, otelConn.Conn)
			}
		})
	}
}

func TestOtDriver_OpenConnector(t *testing.T) {
	testCases := []struct {
		name  string
		error bool
	}{
		{
			name: "no error",
		},
		{
			name:  "with error",
			error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			md := newMockDriver(tc.error)
			d := newDriver(md, config{})

			otelDriver := d.(*otDriver)
			connector, err := otelDriver.OpenConnector("test")

			assert.Equal(t, "test", md.openConnectorName)
			assert.Equal(t, 1, md.openConnectorCount)

			if tc.error {
				assert.Error(t, err)
			} else {
				otelConnector, ok := connector.(*otConnector)
				require.True(t, ok)
				assert.IsType(t, &mockConnector{}, otelConnector.Connector)
			}
		})
	}
}
