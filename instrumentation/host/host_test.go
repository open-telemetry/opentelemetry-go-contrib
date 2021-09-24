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
	gonet "net"
	"os"
	"testing"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/metrictest"

	"go.opentelemetry.io/contrib/instrumentation/host"
)

func getMetric(impl *metrictest.MeterImpl, name string, lbl attribute.KeyValue) float64 {
	for _, b := range impl.MeasurementBatches {
		foundAttribute := false
		for _, haveLabel := range b.Labels {
			if haveLabel != lbl {
				continue
			}
			foundAttribute = true
			break
		}
		if !foundAttribute {
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

func TestHostCPU(t *testing.T) {
	impl, provider := metrictest.NewMeterProvider()
	err := host.Start(
		host.WithMeterProvider(provider),
	)
	assert.NoError(t, err)

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		t.Errorf("could not find this process: %w", err)
	}

	ctx := context.Background()
	processBefore, err := proc.TimesWithContext(ctx)
	require.NoError(t, err)

	hostBefore, err := cpu.TimesWithContext(ctx, false)
	require.NoError(t, err)

	start := time.Now()
	for time.Since(start) < time.Second {
		// This has a mix of user and system time, so serves
		// the purpose of advancing both process and host,
		// user and system CPU usage.
		_, err = proc.TimesWithContext(ctx)
		require.NoError(t, err)
	}

	impl.RunAsyncInstruments()

	processUser := getMetric(impl, "process.cpu.time", host.AttributeCPUTimeUser[0])
	processSystem := getMetric(impl, "process.cpu.time", host.AttributeCPUTimeSystem[0])

	hostUser := getMetric(impl, "system.cpu.time", host.AttributeCPUTimeUser[0])
	hostSystem := getMetric(impl, "system.cpu.time", host.AttributeCPUTimeSystem[0])

	processAfter, err := proc.TimesWithContext(ctx)
	require.NoError(t, err)

	hostAfter, err := cpu.TimesWithContext(ctx, false)
	require.NoError(t, err)

	// Validate process times:
	// User times are in range
	require.LessOrEqual(t, processBefore.User, processUser)
	require.GreaterOrEqual(t, processAfter.User, processUser)
	// System times are in range
	require.LessOrEqual(t, processBefore.System, processSystem)
	require.GreaterOrEqual(t, processAfter.System, processSystem)
	// Ranges are not empty
	require.NotEqual(t, processAfter.System, processBefore.System)
	require.NotEqual(t, processAfter.User, processBefore.User)

	// Validate host times:
	// Correct assumptions:
	require.Equal(t, 1, len(hostBefore))
	require.Equal(t, 1, len(hostAfter))
	// User times are in range
	require.LessOrEqual(t, hostBefore[0].User, hostUser)
	require.GreaterOrEqual(t, hostAfter[0].User, hostUser)
	// System times are in range
	require.LessOrEqual(t, hostBefore[0].System, hostSystem)
	require.GreaterOrEqual(t, hostAfter[0].System, hostSystem)
	// Ranges are not empty
	require.NotEqual(t, hostAfter[0].System, hostBefore[0].System)
	require.NotEqual(t, hostAfter[0].User, hostBefore[0].User)

	// TODO: We are not testing host "Other" nor "Idle" and
	// generally the specification hasn't been finalized, so
	// there's more to do.  Moreover, "Other" is not portable and
	// "Idle" may not advance on a fully loaded machine => both
	// are difficult to test.
}

func TestHostMemory(t *testing.T) {
	impl, provider := metrictest.NewMeterProvider()
	err := host.Start(
		host.WithMeterProvider(provider),
	)
	assert.NoError(t, err)

	ctx := context.Background()
	vMem, err := mem.VirtualMemoryWithContext(ctx)
	require.NoError(t, err)

	impl.RunAsyncInstruments()

	hostUsed := getMetric(impl, "system.memory.usage", host.AttributeMemoryUsed[0])
	assert.Greater(t, hostUsed, 0.0)
	assert.LessOrEqual(t, hostUsed, float64(vMem.Total))

	hostAvailable := getMetric(impl, "system.memory.usage", host.AttributeMemoryAvailable[0])
	assert.GreaterOrEqual(t, hostAvailable, 0.0)
	assert.Less(t, hostAvailable, float64(vMem.Total))

	hostUsedUtil := getMetric(impl, "system.memory.utilization", host.AttributeMemoryUsed[0])
	assert.Greater(t, hostUsedUtil, 0.0)
	assert.LessOrEqual(t, hostUsedUtil, 1.0)

	hostAvailableUtil := getMetric(impl, "system.memory.utilization", host.AttributeMemoryAvailable[0])
	assert.GreaterOrEqual(t, hostAvailableUtil, 0.0)
	assert.Less(t, hostAvailableUtil, 1.0)

	if hostUsed > hostAvailable {
		assert.Greater(t, hostUsedUtil, hostAvailableUtil)
	} else {
		assert.Less(t, hostUsedUtil, hostAvailableUtil)
	}
}

func sendBytes(t *testing.T, count int) error {
	conn1, err := gonet.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer conn1.Close()

	conn2, err := gonet.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer conn2.Close()

	data1 := make([]byte, 1000)
	data2 := make([]byte, 1000)
	for i := range data1 {
		data1[i] = byte(i)
	}

	for ; count > 0; count -= len(data1) {
		_, err = conn1.WriteTo(data1, conn2.LocalAddr())
		if err != nil {
			return err
		}
		_, readAddr, err := conn2.ReadFrom(data2)
		if err != nil {
			return err
		}

		require.Equal(t, "udp", readAddr.Network())
		require.Equal(t, conn1.LocalAddr().String(), readAddr.String())
	}

	return nil
}

func TestHostNetwork(t *testing.T) {
	impl, provider := metrictest.NewMeterProvider()
	err := host.Start(
		host.WithMeterProvider(provider),
	)
	assert.NoError(t, err)

	ctx := context.Background()
	hostBefore, err := net.IOCountersWithContext(ctx, false)
	require.NoError(t, err)

	const howMuch = 10000
	err = sendBytes(t, howMuch)
	require.NoError(t, err)

	// As we are going to read the /proc file system for this info, sleep a while:
	require.Eventually(t, func() bool {
		hostAfter, err := net.IOCountersWithContext(ctx, false)
		require.NoError(t, err)

		return uint64(howMuch) <= hostAfter[0].BytesSent-hostBefore[0].BytesSent &&
			uint64(howMuch) <= hostAfter[0].BytesRecv-hostBefore[0].BytesRecv
	}, 30*time.Second, time.Second/2)

	impl.RunAsyncInstruments()
	hostTransmit := getMetric(impl, "system.network.io", host.AttributeNetworkTransmit[0])
	hostReceive := getMetric(impl, "system.network.io", host.AttributeNetworkReceive[0])

	// Check that the recorded measurements reflect the same change:
	require.LessOrEqual(t, uint64(howMuch), uint64(hostTransmit)-hostBefore[0].BytesSent)
	require.LessOrEqual(t, uint64(howMuch), uint64(hostReceive)-hostBefore[0].BytesRecv)
}
