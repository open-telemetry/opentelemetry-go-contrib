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

package cortex

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/otel/api/global"
	apimetric "go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/export/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

// Exporter forwards metrics to a Cortex instance
type Exporter struct {
	config Config
}

// ExportKindFor returns CumulativeExporter so the Processor correctly aggregates data
func (e *Exporter) ExportKindFor(*apimetric.Descriptor, aggregation.Kind) metric.ExportKind {
	return metric.CumulativeExporter
}

// Export forwards metrics to Cortex from the SDK
func (e *Exporter) Export(_ context.Context, checkpointSet metric.CheckpointSet) error {
	timeseries, err := e.ConvertToTimeSeries(checkpointSet)
	if err != nil {
		return err
	}

	message, buildMessageErr := e.buildMessage(timeseries)
	if buildMessageErr != nil {
		return buildMessageErr
	}

	request, buildRequestErr := e.buildRequest(message)
	if buildRequestErr != nil {
		return buildRequestErr
	}

	sendRequestErr := e.sendRequest(request)
	if sendRequestErr != nil {
		return sendRequestErr
	}

	return nil
}

// NewRawExporter validates the Config struct and creates an Exporter with it.
func NewRawExporter(config Config) (*Exporter, error) {
	// This is redundant when the user creates the Config struct with the NewConfig
	// function.
	if err := config.Validate(); err != nil {
		return nil, err
	}

	exporter := Exporter{config}
	return &exporter, nil
}

// NewExportPipeline sets up a complete export pipeline with a push Controller and
// Exporter.
func NewExportPipeline(config Config, options ...push.Option) (*push.Controller, error) {
	exporter, err := NewRawExporter(config)
	if err != nil {
		return nil, err
	}

	pusher := push.New(
		basic.New(
			simple.NewWithHistogramDistribution(config.HistogramBoundaries),
			exporter,
		),
		exporter,
		options...,
	)
	pusher.Start()
	return pusher, nil
}

// InstallNewPipeline registers a push Controller's Provider globally.
func InstallNewPipeline(config Config, options ...push.Option) (*push.Controller, error) {
	pusher, err := NewExportPipeline(config, options...)
	if err != nil {
		return nil, err
	}
	global.SetMeterProvider(pusher.Provider())
	return pusher, nil
}

// ConvertToTimeSeries converts a CheckpointSet to a slice of TimeSeries pointers
// Based on the aggregation type, ConvertToTimeSeries will call helper functions like
// convertFromSum to generate the correct number of TimeSeries.
func (e *Exporter) ConvertToTimeSeries(checkpointSet export.CheckpointSet) ([]*prompb.TimeSeries, error) {
	var aggError error
	var timeSeries []*prompb.TimeSeries

	// Iterate over each record in the checkpoint set and convert to TimeSeries
	aggError = checkpointSet.ForEach(e, func(record metric.Record) error {
		// Convert based on aggregation type
		agg := record.Aggregation()

		// The following section uses loose type checking to determine how to
		// convert aggregations to timeseries. More "expensive" timeseries are
		// checked first. For example, because a Distribution has a Sum value,
		// we must check for Distribution first or else only the Sum would be
		// converted and the other values like Quantiles would not be.
		//
		// See the Aggregator Kind for more information
		// https://github.com/open-telemetry/opentelemetry-go/blob/master/sdk/export/metric/aggregation/aggregation.go#L123-L138
		if histogram, ok := agg.(aggregation.Histogram); ok {
			tSeries, err := convertFromHistogram(record, histogram)
			if err != nil {
				return err
			}
			timeSeries = append(timeSeries, tSeries...)
		} else if distribution, ok := agg.(aggregation.Distribution); ok && len(e.config.Quantiles) != 0 {
			tSeries, err := convertFromDistribution(record, distribution, e.config.Quantiles)
			if err != nil {
				return err
			}
			timeSeries = append(timeSeries, tSeries...)
		} else if sum, ok := agg.(aggregation.Sum); ok {
			tSeries, err := convertFromSum(record, sum)
			if err != nil {
				return err
			}
			timeSeries = append(timeSeries, tSeries)
			if minMaxSumCount, ok := agg.(aggregation.MinMaxSumCount); ok {
				tSeries, err := convertFromMinMaxSumCount(record, minMaxSumCount)
				if err != nil {
					return err
				}
				timeSeries = append(timeSeries, tSeries...)
			}
		} else if lastValue, ok := agg.(aggregation.LastValue); ok {
			tSeries, err := convertFromLastValue(record, lastValue)
			if err != nil {
				return err
			}
			timeSeries = append(timeSeries, tSeries)
		} else {
			// Report to the user when no conversion was found
			fmt.Printf("No conversion found for record: %s\n", record.Descriptor().Name())
		}

		return nil
	})

	// Check if error was returned in checkpointSet.ForEach()
	if aggError != nil {
		return nil, aggError
	}

	return timeSeries, nil
}

// createTimeSeries is a helper function to create a timeseries from a value and labels
func createTimeSeries(record metric.Record, value apimetric.Number, valueNumberKind apimetric.NumberKind, extraLabels ...label.KeyValue) *prompb.TimeSeries {
	sample := prompb.Sample{
		Value:     value.CoerceToFloat64(valueNumberKind),
		Timestamp: int64(time.Nanosecond) * record.EndTime().UnixNano() / int64(time.Millisecond),
	}

	labels := createLabelSet(record, extraLabels...)

	return &prompb.TimeSeries{
		Samples: []prompb.Sample{sample},
		Labels:  labels,
	}
}

// convertFromSum returns a single TimeSeries based on a Record with a Sum aggregation
func convertFromSum(record metric.Record, sum aggregation.Sum) (*prompb.TimeSeries, error) {
	// Get Sum value
	value, err := sum.Sum()
	if err != nil {
		return nil, err
	}

	// Create TimeSeries. Note that Cortex requires the name label to be in the format
	// "__name__". This is the case for all time series created by this exporter.
	name := sanitize(record.Descriptor().Name())
	numberKind := record.Descriptor().NumberKind()
	tSeries := createTimeSeries(record, value, numberKind, label.String("__name__", name))

	return tSeries, nil
}

// convertFromLastValue returns a single TimeSeries based on a Record with a LastValue aggregation
func convertFromLastValue(record metric.Record, lastValue aggregation.LastValue) (*prompb.TimeSeries, error) {
	// Get value
	value, _, err := lastValue.LastValue()
	if err != nil {
		return nil, err
	}

	// Create TimeSeries
	name := sanitize(record.Descriptor().Name())
	numberKind := record.Descriptor().NumberKind()
	tSeries := createTimeSeries(record, value, numberKind, label.String("__name__", name))

	return tSeries, nil
}

// convertFromMinMaxSumCount returns 4 TimeSeries for the min, max, sum, and count from the mmsc aggregation
func convertFromMinMaxSumCount(record metric.Record, minMaxSumCount aggregation.MinMaxSumCount) ([]*prompb.TimeSeries, error) {
	numberKind := record.Descriptor().NumberKind()

	// Convert Min
	min, err := minMaxSumCount.Min()
	if err != nil {
		return nil, err
	}
	name := sanitize(record.Descriptor().Name() + "_min")
	minTimeSeries := createTimeSeries(record, min, numberKind, label.String("__name__", name))

	// Convert Max
	max, err := minMaxSumCount.Max()
	if err != nil {
		return nil, err
	}
	name = sanitize(record.Descriptor().Name() + "_max")
	maxTimeSeries := createTimeSeries(record, max, numberKind, label.String("__name__", name))

	// Convert Count
	count, err := minMaxSumCount.Count()
	if err != nil {
		return nil, err
	}
	name = sanitize(record.Descriptor().Name() + "_count")
	countTimeSeries := createTimeSeries(record, apimetric.NewInt64Number(count), apimetric.Int64NumberKind, label.String("__name__", name))

	// Return all timeSeries
	tSeries := []*prompb.TimeSeries{
		minTimeSeries, maxTimeSeries, countTimeSeries,
	}

	return tSeries, nil
}

// convertFromDistribution returns len(quantiles) number of TimeSeries in a distribution.
func convertFromDistribution(record metric.Record, distribution aggregation.Distribution, quantiles []float64) ([]*prompb.TimeSeries, error) {
	var timeSeries []*prompb.TimeSeries
	metricName := sanitize(record.Descriptor().Name())
	numberKind := record.Descriptor().NumberKind()

	// Convert Min
	min, err := distribution.Min()
	if err != nil {
		return nil, err
	}
	name := sanitize(metricName + "_min")
	minTimeSeries := createTimeSeries(record, min, numberKind, label.String("__name__", name))
	timeSeries = append(timeSeries, minTimeSeries)

	// Convert Max
	max, err := distribution.Max()
	if err != nil {
		return nil, err
	}
	name = sanitize(metricName + "_max")
	maxTimeSeries := createTimeSeries(record, max, numberKind, label.String("__name__", name))
	timeSeries = append(timeSeries, maxTimeSeries)

	// Convert Sum
	sum, err := distribution.Sum()
	if err != nil {
		return nil, err
	}
	name = sanitize(metricName + "_sum")
	sumTimeSeries := createTimeSeries(record, sum, numberKind, label.String("__name__", name))
	timeSeries = append(timeSeries, sumTimeSeries)

	// Convert Count
	count, err := distribution.Count()
	if err != nil {
		return nil, err
	}
	name = sanitize(record.Descriptor().Name() + "_count")
	countTimeSeries := createTimeSeries(record, apimetric.NewInt64Number(count), apimetric.Int64NumberKind, label.String("__name__", name))
	timeSeries = append(timeSeries, countTimeSeries)

	// For each configured quantile, get the value and create a timeseries
	for _, q := range quantiles {
		value, err := distribution.Quantile(q)
		if err != nil {
			return nil, err
		}

		// Add quantile as a label. e.g. {quantile="0.5"}
		quantileStr := strconv.FormatFloat(q, 'f', -1, 64)

		// Create TimeSeries
		tSeries := createTimeSeries(record, value, numberKind, label.String("__name__", metricName), label.String("quantile", quantileStr))
		timeSeries = append(timeSeries, tSeries)
	}

	return timeSeries, nil
}

// convertFromHistogram returns len(histogram.Buckets) timeseries for a histogram aggregation
func convertFromHistogram(record metric.Record, histogram aggregation.Histogram) ([]*prompb.TimeSeries, error) {
	var timeSeries []*prompb.TimeSeries
	metricName := sanitize(record.Descriptor().Name())
	numberKind := record.Descriptor().NumberKind()

	// Create Sum TimeSeries
	sum, err := histogram.Sum()
	if err != nil {
		return nil, err
	}
	sumTimeSeries := createTimeSeries(record, sum, numberKind, label.String("__name__", metricName+"_sum"))
	timeSeries = append(timeSeries, sumTimeSeries)

	// Handle Histogram buckets
	buckets, err := histogram.Histogram()
	if err != nil {
		return nil, err
	}

	var totalCount float64
	// counts maps from the bucket upper-bound to the cumulative count.
	// The bucket with upper-bound +inf is not included.
	counts := make(map[float64]float64, len(buckets.Boundaries))
	for i, boundary := range buckets.Boundaries {
		// Add bucket count to totalCount and record in map
		totalCount += buckets.Counts[i]
		counts[boundary] = totalCount

		// Add upper boundary as a label. e.g. {le="5"}
		boundaryStr := strconv.FormatFloat(boundary, 'f', -1, 64)

		// Create timeSeries and append
		boundaryTimeSeries := createTimeSeries(record, apimetric.NewFloat64Number(totalCount), apimetric.Float64NumberKind, label.String("__name__", metricName), label.String("le", boundaryStr))
		timeSeries = append(timeSeries, boundaryTimeSeries)
	}

	// Include the +inf boundary in the total count
	totalCount += buckets.Counts[len(buckets.Counts)-1]

	// Create a timeSeries for the +inf bucket and total count
	// These are the same and are both required by Prometheus-based backends

	upperBoundTimeSeries := createTimeSeries(record, apimetric.NewFloat64Number(totalCount), apimetric.Float64NumberKind, label.String("__name__", metricName), label.String("le", "+inf"))

	countTimeSeries := createTimeSeries(record, apimetric.NewFloat64Number(totalCount), apimetric.Float64NumberKind, label.String("__name__", metricName+"_count"))

	timeSeries = append(timeSeries, upperBoundTimeSeries)
	timeSeries = append(timeSeries, countTimeSeries)

	return timeSeries, nil
}

// createLabelSet combines labels from a Record, resource, and extra labels to create a
// slice of prompb.Label.
func createLabelSet(record metric.Record, extraLabels ...label.KeyValue) []*prompb.Label {
	// Map ensure no duplicate label names.
	labelMap := map[string]prompb.Label{}

	// mergeLabels merges Record and Resource labels into a single set, giving precedence
	// to the record's labels.
	mi := label.NewMergeIterator(record.Labels(), record.Resource().LabelSet())
	for mi.Next() {
		label := mi.Label()
		key := string(label.Key)
		labelMap[key] = prompb.Label{
			Name:  sanitize(key),
			Value: label.Value.Emit(),
		}
	}

	// Add extra labels created by the exporter like the metric name or labels to
	// represent histogram buckets.
	for _, label := range extraLabels {
		// Ensure label doesn't exist. If it does, notify user that a user created label
		// is being overwritten by a Prometheus reserved label (e.g. 'le' for histograms)
		key := string(label.Key)
		value := label.Value.AsString()
		_, found := labelMap[key]
		if found {
			log.Printf("Label %s is overwritten. Check if Prometheus reserved labels are used.\n", key)
		}
		labelMap[key] = prompb.Label{
			Name:  key,
			Value: value,
		}
	}

	// Create slice of labels from labelMap and return
	res := make([]*prompb.Label, 0, len(labelMap))
	for _, lb := range labelMap {
		currentLabel := lb
		res = append(res, &currentLabel)
	}

	return res
}

// addHeaders adds required headers, an Authorization header, and all headers in the
// Config Headers map to a http request.
func (e *Exporter) addHeaders(req *http.Request) error {
	// Cortex expects Snappy-compressed protobuf messages. These three headers are
	// hard-coded as they should be on every request.
	req.Header.Add("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.Header.Add("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")

	// Add all user-supplied headers to the request.
	for name, field := range e.config.Headers {
		req.Header.Add(name, field)
	}

	// Add Authorization header if it wasn't already set.
	if _, exists := e.config.Headers["Authorization"]; !exists {
		if err := e.addBearerTokenAuth(req); err != nil {
			return err
		}
		if err := e.addBasicAuth(req); err != nil {
			return err
		}
	}

	return nil
}

// buildMessage creates a Snappy-compressed protobuf message from a slice of TimeSeries.
func (e *Exporter) buildMessage(timeseries []*prompb.TimeSeries) ([]byte, error) {
	// Wrap the TimeSeries as a WriteRequest since Cortex requires it.
	writeRequest := &prompb.WriteRequest{
		Timeseries: timeseries,
	}

	// Convert the struct to a slice of bytes and then compress it.
	message, err := proto.Marshal(writeRequest)
	if err != nil {
		return nil, err
	}
	compressed := snappy.Encode(nil, message)

	return compressed, nil
}

// buildRequest creates an http POST request with a Snappy-compressed protocol buffer
// message as the body and with all the headers attached.
func (e *Exporter) buildRequest(message []byte) (*http.Request, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		e.config.Endpoint,
		bytes.NewBuffer(message),
	)
	if err != nil {
		return nil, err
	}

	// Add the required headers and the headers from Config.Headers.
	err = e.addHeaders(req)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// sendRequest sends an http request using the Exporter's http Client.
func (e *Exporter) sendRequest(req *http.Request) error {
	// Set a client if the user didn't provide one.
	if e.config.Client == nil {
		client, err := e.buildClient()
		if err != nil {
			return err
		}
		e.config.Client = client
	}

	// Attempt to send request.
	res, err := e.config.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// The response should have a status code of 200.
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%v", res.Status)
	}
	return nil
}
