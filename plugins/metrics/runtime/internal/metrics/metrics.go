package metrics

import (
	"context"
	"os"
	"runtime"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

type Metrics struct {
	ProcessCPU ProcessCPU
	CPU        map[string]CPU
	NIC        map[string]NIC
	Memory     Memory
	Runtime    Runtime
}

type Runtime struct {
	NumGC        uint64
	NumGoroutine uint64
}

type ProcessCPU struct {
	User   float64
	System float64
}

type CPU struct {
	User   float64
	System float64
	Usage  float64
	Total  float64
}

type NIC struct {
	BytesReceived uint64
	BytesSent     uint64
}

type Memory struct {
	Available uint64
	Total     uint64
	HeapAlloc uint64
}

func Measure(ctx context.Context) (Metrics, error) {
	p, err := process.NewProcess(int32(os.Getpid())) // TODO: cache the process
	if err != nil {
		return Metrics{}, err
	}

	processTimes, err := p.TimesWithContext(ctx) // returns user and system time for process
	if err != nil {
		return Metrics{}, err
	}

	systemTimes, err := cpu.TimesWithContext(ctx, false)
	if err != nil {
		return Metrics{}, err
	}

	netStats, err := net.IOCountersWithContext(ctx, false)
	if err != nil {
		return Metrics{}, err
	}

	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	runtime := Runtime{
		NumGC:        uint64(rtm.NumGC),
		NumGoroutine: uint64(runtime.NumGoroutine()),
	}

	memStats, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Metrics{}, err
	}

	metrics := Metrics{
		ProcessCPU: ProcessCPU{
			User:   processTimes.User,
			System: processTimes.System,
		},
		CPU: make(map[string]CPU, len(systemTimes)),
		NIC: make(map[string]NIC, len(netStats)),
		Memory: Memory{
			Available: memStats.Available,
			Total:     memStats.Total,
			HeapAlloc: rtm.HeapAlloc,
		},
		Runtime: runtime,
	}

	for _, t := range systemTimes {
		usage := t.User + t.System + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal
		metrics.CPU[t.CPU] = CPU{
			User:   t.User,
			System: t.System,
			Usage:  usage,
			Total:  usage + t.Idle,
		}
	}

	for _, counters := range netStats {
		metrics.NIC[counters.Name] = NIC{
			BytesReceived: counters.BytesRecv,
			BytesSent:     counters.BytesSent,
		}
	}

	return metrics, nil
}
