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

package host_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/internal/metric"
	"go.opentelemetry.io/otel/api/kv"
)

func getMetric(impl *metric.MeterImpl, name string, label kv.KeyValue) float64 {
	for _, b := range impl.MeasurementBatches {
		foundLabel := false
		for _, haveLabel := range b.Labels {
			if haveLabel != label {
				continue
			}
			foundLabel = true
			break
		}
		if !foundLabel {
			continue
		}

		for _, m := range b.Measurements {
			if m.Instrument.Descriptor().Name() != name {
				continue
			}

			return m.Number.CoerceToFloat64(m.Instrument.Descriptor().NumberKind())
		}
	}
	panic("Could not locate a metric in test output")
}

func TestProcessCPU(t *testing.T) {
	impl, provider := metric.NewProvider()
	err := host.Start(
		host.Configure(
			host.WithMeterProvider(provider),
		),
	)
	assert.NoError(t, err)

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		t.Errorf("could not find this process: %w", err)
	}

	ctx := context.Background()
	timesBefore, err := proc.TimesWithContext(ctx)
	require.NoError(t, err)

	start := time.Now()
	for time.Now().Sub(start) < time.Second {
		// This has a mix of user and system time, so serves
		// the purpose of advancing both.
		_, err = proc.TimesWithContext(ctx)
		require.NoError(t, err)
	}

	impl.RunAsyncInstruments()

	processUser := getMetric(impl, "process.cpu.time", host.LabelCPUTimeUser[0])
	processSystem := getMetric(impl, "process.cpu.time", host.LabelCPUTimeSystem[0])

	impl.MeasurementBatches = nil

	timesAfter, err := proc.TimesWithContext(ctx)
	require.NoError(t, err)

	// User times are in range
	require.LessOrEqual(t, timesBefore.User, processUser)
	require.GreaterOrEqual(t, timesAfter.User, processUser)

	// System times are in range
	require.LessOrEqual(t, timesBefore.System, processSystem)
	require.GreaterOrEqual(t, timesAfter.System, processSystem)

	// Ranges are not empty
	require.NotEqual(t, timesAfter.System, timesBefore.System)
	require.NotEqual(t, timesAfter.User, timesBefore.User)
}
