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

package otelsarama // import "go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/unit"
	"go.uber.org/atomic"
)

const (
	saramaKafkaVersion = 2
)

type rateMetric struct {
	startedAt          time.Time
	recordAccumulation atomic.Float32
}

// NewRateMetric returns a rate metric to be used for calculation of per second average.
func newRateMetric() rateMetric {
	return rateMetric{
		startedAt:          time.Now(),
		recordAccumulation: *atomic.NewFloat32(0),
	}
}

func (m *rateMetric) Add(record float64) {
	m.recordAccumulation.Add(float32(record))
}

func (m *rateMetric) Average() float64 {
	secondElapsed := time.Since(m.startedAt).Seconds()
	loaded := float64(m.recordAccumulation.Load())

	// flush all measure units
	m.startedAt = time.Now()
	m.recordAccumulation.Swap(0)

	return loaded / secondElapsed
}

// PRODUCER METRICS:
// Implementation of producer metrics defined in otel specification
type producerOutgoingBytesRate struct {
	rateRecorder rateMetric
	metric       asyncfloat64.Gauge
}

type producerMeters struct {
	producerOutgoingBytesRate producerOutgoingBytesRate //messaging.kafka.producer.outgoing-bytes.rate
}

func newProducerMeters(meter metric.Meter) producerMeters {
	var (
		pm  producerMeters = producerMeters{producerOutgoingBytesRate: producerOutgoingBytesRate{}}
		err error
	)

	if pm.producerOutgoingBytesRate.metric, err = meter.AsyncFloat64().Gauge(
		"messaging.kafka.producer.outgoing-bytes.rate",
		instrument.WithUnit(unit.Bytes),
	); err != nil {
		otel.Handle(err)
	}
	pm.producerOutgoingBytesRate.rateRecorder = newRateMetric()

	return pm
}

func (pmeter *producerMeters) ObserveProducerOutgoingBytesRate(ctx context.Context, attrs ...attribute.KeyValue) {
	avg := pmeter.producerOutgoingBytesRate.rateRecorder.Average()
	pmeter.producerOutgoingBytesRate.metric.Observe(ctx, avg, attrs...)
}
