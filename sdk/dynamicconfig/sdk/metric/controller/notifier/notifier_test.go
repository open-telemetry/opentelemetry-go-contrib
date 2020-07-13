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

package notifier_test

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notify "go.opentelemetry.io/contrib/sdk/dynamicconfig/sdk/metric/controller/notifier"
	controllerTest "go.opentelemetry.io/otel/sdk/metric/controller/test"
)

// testLock is to prevent race conditions in test code
// testVar is used to verify OnInitialConfig and OnUpdatedConfig are called
type testWatcher struct {
	testLock sync.Mutex
	testVar  int
}

func (w *testWatcher) OnInitialConfig(config *notify.Config) error {
	w.testLock.Lock()
	defer w.testLock.Unlock()
	w.testVar = 1
	return nil
}

func (w *testWatcher) OnUpdatedConfig(config *notify.Config) error {
	w.testLock.Lock()
	defer w.testLock.Unlock()
	w.testVar = 2
	return nil
}

// Use a getter to prevent race conditions around testVar
func (w *testWatcher) getTestVar() int {
	w.testLock.Lock()
	defer w.testLock.Unlock()
	return w.testVar
}

func newExampleNotifier(t *testing.T) *notify.Notifier {
	notifier, err := notify.NewNotifier(
		notify.GetDefaultConfig(1, []byte{'b', 'a', 'r'}),
		notify.WithCheckFrequency(time.Minute),
		notify.WithConfigHost(notify.TestAddress),
		notify.WithResource(notify.MockResource("notifiertest")),
	)
	assert.NoError(t, err)

	return notifier
}

// Test config updates
func TestDynamicNotifier(t *testing.T) {
	watcher := testWatcher{
		testVar: 0,
	}
	mock := controllerTest.NewMockClock()

	stopFunc := notify.RunMockConfigService(
		t,
		notify.TestAddress,
		notify.GetDefaultConfig(1, notify.TestFingerprint),
	)
	defer stopFunc()

	notifier := newExampleNotifier(t)
	require.Equal(t, 0, watcher.getTestVar())

	notifier.SetClock(mock)
	notifier.Start()
	defer notifier.Stop()

	notifier.Register(&watcher)
	require.Equal(t, 1, watcher.getTestVar())

	mock.Add(5 * time.Minute)
	runtime.Gosched()

	require.Equal(t, 2, watcher.getTestVar())
}

// Test config doesn't update
func TestNonDynamicNotifier(t *testing.T) {
	watcher := testWatcher{
		testVar: 0,
	}
	mock := controllerTest.NewMockClock()
	notifier, err := notify.NewNotifier(
		notify.GetDefaultConfig(60, notify.TestFingerprint),
	)
	assert.NoError(t, err)
	require.Equal(t, 0, watcher.getTestVar())

	notifier.SetClock(mock)
	notifier.Start()
	defer notifier.Stop()

	notifier.Register(&watcher)
	require.Equal(t, 1, watcher.getTestVar())

	mock.Add(time.Minute)

	require.Equal(t, 1, watcher.getTestVar())
}

func TestDoubleStop(t *testing.T) {
	stopFunc := notify.RunMockConfigService(
		t,
		notify.TestAddress,
		notify.GetDefaultConfig(60, notify.TestFingerprint),
	)
	defer stopFunc()
	notifier := newExampleNotifier(t)
	notifier.Start()
	notifier.Stop()
	notifier.Stop()
}

func TestPushDoubleStart(t *testing.T) {
	stopFunc := notify.RunMockConfigService(
		t,
		notify.TestAddress,
		notify.GetDefaultConfig(60, notify.TestFingerprint),
	)
	defer stopFunc()
	notifier := newExampleNotifier(t)
	notifier.Start()
	notifier.Start()
	notifier.Stop()
}
