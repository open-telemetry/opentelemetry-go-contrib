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
	"errors"
	"fmt"
	"sync"

	"github.com/rcrowley/go-metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/unit"
)

type metricsType string

const (
	HISTOGRAM metricsType = "histogram"
	GAUGE     metricsType = "gauge"
	COUNTER   metricsType = "counter"
)

type variableType string

const (
	INT64   variableType = "int64"
	FLOAT64 variableType = "float64"
)

type metricsProps struct {
	Name              string
	MetricType        metricsType
	MetricUnit        unit.Unit
	Description       string
	VariableType      variableType
	RetrievalFunction func(r metrics.Registry) interface{}
}

const (
	metricsReservoirSize = 1028
	metricsAlphaFactor   = 0.015
)

type observable[T int64 | float64] interface {
	instrument.Asynchronous
	Observe(ctx context.Context, x T, attrs ...attribute.KeyValue)
}

func producerMetrics( /*topics []string*/ ) []metricsProps {
	return []metricsProps{
		{
			Name:       "batch-size",
			MetricType: GAUGE,
			MetricUnit: unit.Bytes,
			// derived from sarama doc
			Description:  "Distribution of the number of bytes sent per partition per request for all topics",
			VariableType: FLOAT64,
			RetrievalFunction: func(r metrics.Registry) interface{} {
				var val int64
				hist, ok := r.Get("batch-size").(metrics.Histogram)
				if !ok {
					return val
				}
				return hist.Mean()
			}, // TODO:
		}, // -for-topic-<topic>
		{
			Name:       "record-send-rate",
			MetricType: GAUGE,
			MetricUnit: unit.Dimensionless,
			// derived from sarama doc
			Description:  "Records/second sent to all topics",
			VariableType: INT64,
			RetrievalFunction: func(r metrics.Registry) interface{} {
				var val int64
				gaug, ok := r.Get("record-send-rate").(metrics.Gauge)
				if !ok {
					return val
				}
				return gaug.Value()
			},
		}, // -for-topic-<topic>
		{
			Name:       "records-per-request",
			MetricType: GAUGE,
			MetricUnit: unit.Dimensionless,
			// derived from sarama doc
			Description:  "Distribution of the number of records sent per request for all topics",
			VariableType: FLOAT64,
			RetrievalFunction: func(r metrics.Registry) interface{} {
				var val float64
				hist, ok := r.Get("records-per-request").(metrics.Histogram)
				if !ok {
					return val
				}
				return hist.Mean()
			},
		}, // -for-topic-<topic>
		{
			Name:       "compression-ratio",
			MetricType: GAUGE,
			MetricUnit: unit.Dimensionless,
			// derived from sarama doc
			Description:  "Distribution of the compression ratio times 100 of record batches for all topics",
			VariableType: FLOAT64,
			RetrievalFunction: func(r metrics.Registry) interface{} {
				var val float64
				hist, ok := r.Get("compression-ratio").(metrics.Histogram)
				if !ok {
					return val
				}
				return hist.Mean()
			},
		}, // -for-topic-<topic>
	}
}

/*
consumer-batch-size				histogram
consumer-fetch-rate				meter
consumer-fetch-response-size	histogram
*/
func consumerMetrics( /*topics []string*/ ) []metricsProps {
	return []metricsProps{
		{
			Name:       "consumer-batch-size",
			MetricType: GAUGE,
			MetricUnit: unit.Bytes,
			// derived from sarama doc
			Description:  "Distribution of the number of messages in a batch",
			VariableType: FLOAT64,
			RetrievalFunction: func(r metrics.Registry) interface{} {
				var val int64
				hist, ok := r.Get("consumer-batch-size").(metrics.Histogram)
				if !ok {
					return val
				}
				return hist.Mean()
			},
		}, // -for-topic-<topic>
		{
			Name:       "consumer-fetch-rate",
			MetricType: GAUGE,
			MetricUnit: unit.Dimensionless,
			// derived from sarama doc
			Description:  "Fetch requests/second sent to all brokers",
			VariableType: INT64,
			RetrievalFunction: func(r metrics.Registry) interface{} {
				var val int64
				gaug, ok := r.Get("consumer-fetch-rate").(metrics.Gauge)
				if !ok {
					return val
				}
				return gaug.Value()
			},
		}, // -for-topic-<topic>
		{
			Name:       "consumer-fetch-response-size",
			MetricType: GAUGE,
			MetricUnit: unit.Bytes,
			// derived from sarama doc
			Description:  "Distribution of the fetch response size in bytes",
			VariableType: FLOAT64,
			RetrievalFunction: func(r metrics.Registry) interface{} {
				var val float64
				hist, ok := r.Get("consumer-fetch-response-size").(metrics.Histogram)
				if !ok {
					return val
				}
				return hist.Mean()
			},
		},
	}
}

func startProducerMetric(meter metric.Meter, registry metrics.Registry) error {
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()

	if registry == nil {
		return nil
	}

	producerMetrics := producerMetrics()

	return startMetrics(meter, registry, producerMetrics)
}

func startConsumerMetric(meter metric.Meter, registry metrics.Registry) error {
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()

	if registry == nil {
		return nil
	}

	consumerMetrics := consumerMetrics()

	return startMetrics(meter, registry, consumerMetrics)
}

func startMetrics(meter metric.Meter, registry metrics.Registry, mets []metricsProps) error {
	var (
		asyncInsts []instrument.Asynchronous         = make([]instrument.Asynchronous, len(mets))
		callbacks  []func(ctx context.Context) error = make([]func(ctx context.Context) error, len(mets))
	)

	for _, met := range mets {
		switch met.VariableType {
		case INT64:
			//Idea: Refinement based on generic intrumentprovider (opentelemetry-go)
			prov := meter.AsyncInt64()
			obs, callback, err := convertToInt64MetricType(prov, registry, met)
			if err != nil {
				return err
			}
			asyncInsts = append(asyncInsts, obs)
			callbacks = append(callbacks, callback)
		case FLOAT64:
			//Idea: Refinement based on generic intrumentprovider (opentelemetry-go)
			prov := meter.AsyncFloat64()
			obs, callback, err := convertToFloat64MetricType(prov, registry, met)
			if err != nil {
				return err
			}
			asyncInsts = append(asyncInsts, obs)
			callbacks = append(callbacks, callback)
		}
	}

	err := meter.RegisterCallback(asyncInsts, func(ctx context.Context) {
		for _, callback := range callbacks {
			if callback != nil { // in initial startup, nil is being returned
				err := callback(ctx)
				if err != nil {
					otel.Handle(err)
				}
			}
		}
	})

	return err
}

func convertToInt64MetricType(prov asyncint64.InstrumentProvider, r metrics.Registry, prop metricsProps) (instrument.Asynchronous, func(ctx context.Context) error, error) {
	var (
		err     error
		metType observable[int64]
	)

	switch prop.MetricType {
	case HISTOGRAM:
		return nil, nil, errors.New("Histogram on Async instrument provier is not supported") // TODO: aggregate error functions for sake of errors.as / is
	case GAUGE:
		metType, err = prov.Gauge(
			prop.Name,
			instrument.WithUnit(prop.MetricUnit),
			instrument.WithDescription(prop.Description),
		)
		if err != nil {
			return metType, nil, err
		}
		break
	case COUNTER:
		metType, err = prov.Counter(
			prop.Name,
			instrument.WithUnit(prop.MetricUnit),
			instrument.WithDescription(prop.Description),
		)
		if err != nil {
			return metType, nil, err
		}
		break
	}

	if metType == nil {
		return metType, nil, errors.New("no metric type found")
	}

	callback := func(ctx context.Context) error {
		val, ok := prop.RetrievalFunction(r).(int64)

		if !ok {
			return fmt.Errorf("RetrievalFunction of %s does not return correct variable type", prop.Name)
		}
		metType.Observe(ctx, val)
		return nil
	}

	return metType, callback, err
}

func convertToFloat64MetricType(prov asyncfloat64.InstrumentProvider, r metrics.Registry, prop metricsProps) (instrument.Asynchronous, func(ctx context.Context) error, error) {
	var (
		err     error
		metType observable[float64]
	)

	switch prop.MetricType {
	case HISTOGRAM:
		return nil, nil, errors.New("Histogram on Async instrument provier is not supported") // TODO: aggregate error functions for sake of errors.as / is
	case GAUGE:
		metType, err = prov.Gauge(
			prop.Name,
			instrument.WithUnit(prop.MetricUnit),
			instrument.WithDescription(prop.Description),
		)
		if err != nil {
			return metType, nil, err
		}
		break
	case COUNTER:
		metType, err = prov.Counter(
			prop.Name,
			instrument.WithUnit(prop.MetricUnit),
			instrument.WithDescription(prop.Description),
		)
		if err != nil {
			return metType, nil, err
		}
		break
	}

	if metType == nil {
		return metType, nil, errors.New("no metric type found")
	}

	callback := func(ctx context.Context) error {
		val, ok := prop.RetrievalFunction(r).(float64)

		if !ok {
			return fmt.Errorf("RetrievalFunction of %s does not return correct variable type", prop.Name)
		}
		metType.Observe(ctx, val)
		return nil
	}

	return metType, callback, err
}
