package datadog_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/DataDog/sketches-go/ddsketch"

	"go.opentelemetry.io/contrib/exporters/metric/datadog"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	integrator "go.opentelemetry.io/otel/sdk/metric/integrator/simple"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

type TestUDPServer struct {
	*net.UDPConn
}

func ExampleExporter() {
	selector := simple.NewWithSketchDistribution(ddsketch.NewDefaultConfig())
	myIntegrator := integrator.New(selector, false)
	exp, err := datadog.NewExporter(datadog.Options{
		Tags:          []string{"env:dev"},
		StatsDOptions: []statsd.Option{statsd.WithoutTelemetry()},
	})
	if err != nil {
		panic(err)
	}
	s, err := getTestServer(datadog.DefaultStatsAddrUDP)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	go func() {
		defer exp.Close()
		pusher := push.New(myIntegrator, exp, push.WithPeriod(time.Second*60))
		defer pusher.Stop()
		pusher.Start()
		global.SetMeterProvider(pusher.Provider())
		meter := global.Meter("marwandist")
		m := metric.Must(meter).NewInt64ValueRecorder("myrecorder")
		meter.RecordBatch(context.Background(), []kv.KeyValue{kv.Int("l", 1)},
			m.Measurement(1), m.Measurement(50), m.Measurement(100))
	}()

	statsChan := make(chan []byte, 1)
	timedOutChan, stopChan := make(chan struct{}), make(chan struct{})
	defer close(stopChan)

	go s.ReadPackets(statsChan, 500*time.Millisecond, timedOutChan, stopChan)

	for {
		select {
		case d := <-statsChan:
			// only look for "max" value, since we don't want to rely on
			// specifics of OpenTelemetry aggregator calculations
			// "max" is something that will always exist and always be the same
			statLine := string(d)
			if strings.HasPrefix(statLine, "myrecorder.max") {
				fmt.Println(statLine)
				return
			}
		case <-timedOutChan:
			_, _ = fmt.Fprintln(os.Stderr, "Server timed out waiting for packets")
			return
		case <-time.After(1 * time.Second):
			fmt.Println("no data received after 1 second")
			return
		}
	}

	// Output:
	// myrecorder.max:100|g|#env:dev,l:1
	//
}

func getTestServer(addr string) (*TestUDPServer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return &TestUDPServer{conn}, nil
}

// ReadPackets reads one "transmission" at a time from the UDP connection
//
// In the case of StatsD, there is one transmission per stat.
// If there is nothing on the connection for longer than maxIdleTime, the
// routine will return, assuming that everything has been sent
// doneChan is an output channel that is closed when ReadPackets returns
// stopChan is an input channel that tells ReadPackets to exit
func (s TestUDPServer) ReadPackets(
	xferChan chan []byte,
	maxIdleTime time.Duration,
	doneChan chan<- struct{},
	stopChan <-chan struct{}) {

	const readTimeout = 50 * time.Millisecond
	var timeouts int

	buffer := make([]byte, 1500)
	defer close(doneChan)
	n := 0
	for {
		select {
		case <-stopChan:
			return
		default:
			_ = s.SetReadDeadline(time.Now().Add(readTimeout))
			nn, _, err := s.ReadFrom(buffer[n:])
			if err == nil {
				timeouts = 0
				data := make([]byte, nn)
				_ = copy(data, buffer[n:n+nn])
				xferChan <- data
				n += nn
				continue
			} else {
				if nerr, ok := err.(*net.OpError); ok && nerr.Timeout() {
					timeouts++
					if time.Duration(timeouts)*readTimeout > maxIdleTime {
						close(doneChan)
						return
					}
					continue
				}
				// give connection some time to connect
				time.Sleep(2 * time.Millisecond)
			}

		}
	}
}
