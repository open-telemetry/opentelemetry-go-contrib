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

func TestContainerIDFromReader(t *testing.T) {
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
			got, err := containerIDFromReader(strings.NewReader(tt.content))
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainerID(t *testing.T) {
	_, err := ContainerID()
	require.NoError(t, err)
}

func TestContainerIDFromMountInfo(t *testing.T) {
	const containerID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name      string
		mountInfo string
		want      string
	}{
		{
			name: "private cgroup namespace",
			mountInfo: "34 25 8:1 /docker/containers/" + containerID +
				"/hostname /etc/hostname rw,relatime - ext4 /dev/sda1 rw\n",
			want: containerID,
		},
		{
			name:      "unrelated hostname mount",
			mountInfo: "34 25 8:1 /etc/hostname /etc/hostname rw,relatime - ext4 /dev/sda1 rw\n",
		},
		{
			name:      "hostname ID on another mount point",
			mountInfo: "34 25 8:1 /docker/containers/" + containerID + "/hostname /mnt/hostname rw,relatime - ext4 /dev/sda1 rw\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := containerIDFromMountInfo(strings.NewReader(tt.mountInfo))
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainerIDFromFiles_PrivateCgroupNamespace(t *testing.T) {
	const containerID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	calls := make([]string, 0, 2)

	got, err := containerIDFromFiles(func(path string) (io.ReadCloser, error) {
		calls = append(calls, path)
		switch path {
		case cgroupPath:
			return io.NopCloser(strings.NewReader("0::/\n")), nil
		case mountInfoPath:
			return io.NopCloser(strings.NewReader(
				"34 25 8:1 /docker/containers/" + containerID + "/hostname /etc/hostname rw,relatime - ext4 /dev/sda1 rw\n",
			)), nil
		default:
			return nil, os.ErrNotExist
		}
	})

	require.NoError(t, err)
	assert.Equal(t, containerID, got)
	assert.Equal(t, []string{cgroupPath, mountInfoPath}, calls)
}

func TestContainerIDFromFiles_CgroupID(t *testing.T) {
	const containerID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	calls := make([]string, 0, 1)

	got, err := containerIDFromFiles(func(path string) (io.ReadCloser, error) {
		calls = append(calls, path)
		return io.NopCloser(strings.NewReader("1:name=systemd:/docker/" + containerID + "\n")), nil
	})

	require.NoError(t, err)
	assert.Equal(t, containerID, got)
	assert.Equal(t, []string{cgroupPath}, calls)
}

func TestContainerIDFromFiles_NoID(t *testing.T) {
	got, err := containerIDFromFiles(func(path string) (io.ReadCloser, error) {
		switch path {
		case cgroupPath:
			return io.NopCloser(strings.NewReader("0::/\n")), nil
		case mountInfoPath:
			return io.NopCloser(strings.NewReader("34 25 8:1 /etc/hostname /etc/hostname rw - ext4 /dev/sda1 rw\n")), nil
		default:
			return nil, os.ErrNotExist
		}
	})

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestContainerIDFromFiles_FilesNotExist(t *testing.T) {
	got, err := containerIDFromFiles(func(string) (io.ReadCloser, error) {
		return nil, os.ErrNotExist
	})

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestContainerIDFromFiles_OpenError(t *testing.T) {
	cgroupErr := errors.New("cgroup read failed")
	mountInfoErr := errors.New("mountinfo read failed")

	got, err := containerIDFromFiles(func(path string) (io.ReadCloser, error) {
		switch path {
		case cgroupPath:
			return nil, cgroupErr
		case mountInfoPath:
			return nil, mountInfoErr
		default:
			return nil, os.ErrNotExist
		}
	})

	assert.Empty(t, got)
	assert.ErrorIs(t, err, cgroupErr)
	assert.ErrorIs(t, err, mountInfoErr)
}
