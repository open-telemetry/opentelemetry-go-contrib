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

	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func getMetric[N float64 | int64](resourceMetrics *metricdata.ResourceMetrics, name string, kv attribute.KeyValue) N {
	//iterDataPoint filter the dataPoint array by attribute
	iterDataPoint := func(dataPoint interface{}, kv attribute.KeyValue) interface{} {
		var dataValue interface{}
		if v, ok := dataPoint.([]metricdata.DataPoint[float64]); ok {
			for _, data := range v {
				if v, ok := data.Attributes.Value(kv.Key); ok && v == kv.Value {
					dataValue = data.Value
				}
			}
		}

		if v, ok := dataPoint.([]metricdata.DataPoint[int64]); ok {
			for _, data := range v {
				if v, ok := data.Attributes.Value(kv.Key); ok && v == kv.Value {
					dataValue = data.Value
				}
			}
		}

		return dataValue
	}

	var dataValue interface{}
	for _, r := range resourceMetrics.ScopeMetrics[0].Metrics {
		if r.Name != name {
			continue
		}
		switch v := r.Data.(type) {
		case metricdata.Sum[float64]:
			dataValue = iterDataPoint(v.DataPoints, kv)
		case metricdata.Sum[int64]:
			dataValue = iterDataPoint(v.DataPoints, kv)
		case metricdata.Gauge[float64]:
			dataValue = iterDataPoint(v.DataPoints, kv)
		case metricdata.Gauge[int64]:
			dataValue = iterDataPoint(v.DataPoints, kv)
		default:
			panic("invalid Aggregation")
		}
	}
	return dataValue.(N)
}
func TestCPU(t *testing.T) {
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
	err := host.Start(
		host.WithMeterProvider(meterProvider),
	)
	assert.NoError(t, err)

	proc, err := process.NewProcess(int32(os.Getpid()))
	require.NoError(t, err)

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

	//Collect metrics
	resourceMetrics := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(ctx, &resourceMetrics))
	//b, _ := json.Marshal(resourceMetrics)
	// todo	fmt.Println(string(b))
	processAfter, err := proc.TimesWithContext(ctx)
	require.NoError(t, err)

	hostAfter, err := cpu.TimesWithContext(ctx, false)
	require.NoError(t, err)

	processUser := getMetric[float64](&resourceMetrics, "process.cpu.time", host.AttributeCPUTimeUser[0])
	processSystem := getMetric[float64](&resourceMetrics, "process.cpu.time", host.AttributeCPUTimeSystem[0])

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

	systemUser := getMetric[float64](&resourceMetrics, "system.cpu.time", host.AttributeCPUTimeUser[0])
	systemSystem := getMetric[float64](&resourceMetrics, "system.cpu.time", host.AttributeCPUTimeSystem[0])

	// Validate host times:
	// Correct assumptions:
	require.Equal(t, 1, len(hostBefore))
	require.Equal(t, 1, len(hostAfter))
	// User times are in range
	require.LessOrEqual(t, hostBefore[0].User, systemUser)
	require.GreaterOrEqual(t, hostAfter[0].User, systemUser)
	// System times are in range
	require.LessOrEqual(t, hostBefore[0].System, systemSystem)
	require.GreaterOrEqual(t, hostAfter[0].System, systemSystem)
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
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
	err := host.Start(
		host.WithMeterProvider(meterProvider),
	)
	assert.NoError(t, err)

	ctx := context.Background()
	vMem, err := mem.VirtualMemoryWithContext(ctx)
	require.NoError(t, err)

	//Collect metrics
	resourceMetrics := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(ctx, &resourceMetrics))

	hostUsed := getMetric[int64](&resourceMetrics, "system.memory.usage", host.AttributeMemoryUsed[0])
	assert.Greater(t, hostUsed, int64(0))
	assert.LessOrEqual(t, hostUsed, int64(vMem.Total))

	hostAvailable := getMetric[int64](&resourceMetrics, "system.memory.usage", host.AttributeMemoryAvailable[0])
	assert.GreaterOrEqual(t, hostAvailable, int64(0))
	assert.Less(t, hostAvailable, int64(vMem.Total))

	hostUsedUtil := getMetric[float64](&resourceMetrics, "system.memory.utilization", host.AttributeMemoryUsed[0])
	assert.Greater(t, hostUsedUtil, 0.0)
	assert.LessOrEqual(t, hostUsedUtil, 1.0)

	hostAvailableUtil := getMetric[float64](&resourceMetrics, "system.memory.utilization", host.AttributeMemoryAvailable[0])
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
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
	err := host.Start(
		host.WithMeterProvider(meterProvider),
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

	// Collect metrics
	resourceMetrics := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(ctx, &resourceMetrics))

	hostTransmit := getMetric[int64](&resourceMetrics, "system.network.io", host.AttributeNetworkTransmit[0])
	hostReceive := getMetric[int64](&resourceMetrics, "system.network.io", host.AttributeNetworkReceive[0])

	// Check that the recorded measurements reflect the same change:
	require.LessOrEqual(t, uint64(howMuch), uint64(hostTransmit)-hostBefore[0].BytesSent)
	require.LessOrEqual(t, uint64(howMuch), uint64(hostReceive)-hostBefore[0].BytesRecv)
}
