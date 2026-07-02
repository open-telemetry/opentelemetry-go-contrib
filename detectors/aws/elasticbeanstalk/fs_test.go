// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package elasticbeanstalk

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEBFileSystem_Open(t *testing.T) {
	path := filepath.Join(t.TempDir(), "environment.conf")
	require.NoError(t, os.WriteFile(path, []byte("contents"), 0o600))

	fs := &ebFileSystem{}
	f, err := fs.Open(path)
	require.NoError(t, err)
	defer f.Close()

	got, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "contents", string(got))
}

func TestEBFileSystem_Open_NotFound(t *testing.T) {
	fs := &ebFileSystem{}
	_, err := fs.Open(filepath.Join(t.TempDir(), "missing.conf"))
	assert.Error(t, err)
}

func TestEBFileSystem_IsWindows(t *testing.T) {
	fs := &ebFileSystem{}
	assert.Equal(t, runtime.GOOS == "windows", fs.IsWindows())
}
