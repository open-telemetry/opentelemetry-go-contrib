// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerID(t *testing.T) {
	const containerID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "cgroup v1",
			content: "1:name=systemd:/docker/" + containerID + "\n",
			want:    containerID,
		},
		{
			name:    "cgroup v2",
			content: "0::/system.slice/docker-" + containerID + ".scope\n",
			want:    containerID,
		},
		{
			name:    "no container ID",
			content: "0::/user.slice/user-1000.slice/session-1.scope\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFileHooks(t, tt.content, nil, nil)

			got, err := ContainerID()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainerID_MissingCgroupFile(t *testing.T) {
	openCalled := false
	previousOSStat := osStat
	previousOSOpen := osOpen
	osStat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	osOpen = func(string) (io.ReadCloser, error) {
		openCalled = true
		return nil, errors.New("unexpected open")
	}
	t.Cleanup(func() {
		osStat = previousOSStat
		osOpen = previousOSOpen
	})

	got, err := ContainerID()
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.False(t, openCalled)
}

func TestContainerID_OpenError(t *testing.T) {
	openErr := errors.New("open failed")
	setFileHooks(t, "", nil, openErr)

	got, err := ContainerID()
	assert.ErrorIs(t, err, openErr)
	assert.Empty(t, got)
}

func setFileHooks(t *testing.T, content string, statErr, openErr error) {
	t.Helper()
	previousOSStat := osStat
	previousOSOpen := osOpen
	osStat = func(string) (os.FileInfo, error) { return nil, statErr }
	osOpen = func(string) (io.ReadCloser, error) {
		if openErr != nil {
			return nil, openErr
		}
		return io.NopCloser(strings.NewReader(content)), nil
	}
	t.Cleanup(func() {
		osStat = previousOSStat
		osOpen = previousOSOpen
	})
}
