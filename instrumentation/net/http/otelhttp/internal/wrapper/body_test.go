// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wrapper

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBodyRead(t *testing.T) {
	b := Body{
		ReadCloser: io.NopCloser(strings.NewReader("Hello, world!")),
		OnRecordFn: func(int64) {},
	}
	c, err := io.ReadAll(&b)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", string(c))
	assert.Equal(t, int64(13), b.ReadLength())
}

func TestBodyOnRecordFn(t *testing.T) {
	var sizeFromFn int64

	b := Body{
		ReadCloser: io.NopCloser(strings.NewReader("Hello, world!")),
		OnRecordFn: func(n int64) {
			sizeFromFn += n
		},
	}
	_, err := io.ReadAll(&b)
	assert.NoError(t, err)
	assert.Equal(t, int64(13), sizeFromFn)
}

func TestBodyReadParallel(t *testing.T) {
	b := Body{
		ReadCloser: io.NopCloser(strings.NewReader("Hello, world!")),
		OnRecordFn: func(int64) {},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.ReadLength()
	}()
	_, err := io.ReadAll(&b)
	assert.NoError(t, err)

	wg.Wait()
}
