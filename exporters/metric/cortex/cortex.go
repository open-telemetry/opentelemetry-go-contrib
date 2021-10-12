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

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/otel/attribute"
	apimetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/sdk/export/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Exporter forwards metrics to a Cortex instance
type Exporter struct {
	config Config
}

type exportData struct {
	export.Record

	Resource *resource.Resource
}

// ExportKindFor returns CumulativeExporter so the Processor correctly aggregates data
func (e *Exporter) ExportKindFor(*apimetric.Descriptor, aggregation.Kind) metric.ExportKind {
	return metric.CumulativeExportKind
}

// Export forwards metrics to Cortex from the SDK
func (e *Exporter) Export(_ context.Context, res *resource.Resource, checkpointSet metric.InstrumentationLibraryReader) error {
	timeseries, err := e.ConvertToTimeSeries(res, checkpointSet)
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
func NewExportPipeline(config Config, options ...controller.Option) (*controller.Controller, error) {
	exporter, err := NewRawExporter(config)
	if err != nil {
		return nil, err
	}

	cont := controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.HistogramBoundaries),
			),
			exporter,
		),
		append(options, controller.WithExporter(exporter))...,
	)

	return cont, cont.Start(context.TODO())
}

// InstallNewPipeline registers a push Controller's MeterProvider globally.
func InstallNewPipeline(config Config, options ...controller.Option) (*controller.Controller, error) {
	cont, err := NewExportPipeline(config, options...)
	if err != nil {
		return nil, err
	}
	global.SetMeterProvider(cont)
	return cont, nil
}

// ConvertToTimeSeries converts a InstrumentationLibraryReader to a slice of TimeSeries pointers
// Based on the aggregation type, ConvertToTimeSeries will call helper functions like
// convertFromSum to generate the correct number of TimeSeries.
func (e *Exporter) ConvertToTimeSeries(res *resource.Resource, checkpointSet export.InstrumentationLibraryReader) ([]prompb.TimeSeries, error) {
	var aggError error
	var timeSeries []prompb.TimeSeries

	// Iterate over each record in the checkpoint set and convert to TimeSeries
	aggError = checkpointSet.ForEach(func(library instrumentation.Library, reader export.Reader) error {
		return reader.ForEach(e, func(record metric.Record) error {
			// Convert based on aggregation type
			edata := exportData{
				Resource: res,
				Record:   record,
			}
			agg := record.Aggregation()

			// The following section uses loose type checking to determine how to
			// convert aggregations to timeseries. More "expensive" timeseries are
			// checked first.
			//
			// See the Aggregator Kind for more information
			// https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/export/metric/aggregation/aggregation.go#L123-L138
			if histogram, ok := agg.(aggregation.Histogram); ok {
				tSeries, err := convertFromHistogram(edata, histogram)
				if err != nil {
					return err
				}
				timeSeries = append(timeSeries, tSeries...)
			} else if sum, ok := agg.(aggregation.Sum); ok {
				tSeries, err := convertFromSum(edata, sum)
				if err != nil {
					return err
				}
				timeSeries = append(timeSeries, tSeries)
				if minMaxSumCount, ok := agg.(aggregation.MinMaxSumCount); ok {
					tSeries, err := convertFromMinMaxSumCount(edata, minMaxSumCount)
					if err != nil {
						return err
					}
					timeSeries = append(timeSeries, tSeries...)
				}
			} else if lastValue, ok := agg.(aggregation.LastValue); ok {
				tSeries, err := convertFromLastValue(edata, lastValue)
				if err != nil {
					return err
				}
				timeSeries = append(timeSeries, tSeries)
			} else {
				// Report to the user when no conversion was found
				fmt.Printf("No conversion found for record: %s\n", edata.Descriptor().Name())
			}

			return nil
		})
	})

	// Check if error was returned in checkpointSet.ForEach()
	if aggError != nil {
		return nil, aggError
	}

	return timeSeries, nil
}

// createTimeSeries is a helper function to create a timeseries from a value and attributes
func createTimeSeries(edata exportData, value number.Number, valueNumberKind number.Kind, extraAttributes ...attribute.KeyValue) prompb.TimeSeries {
	sample := prompb.Sample{
		Value:     value.CoerceToFloat64(valueNumberKind),
		Timestamp: int64(time.Nanosecond) * edata.EndTime().UnixNano() / int64(time.Millisecond),
	}

	attributes := createLabelSet(edata, extraAttributes...)

	return prompb.TimeSeries{
		Samples: []prompb.Sample{sample},
		Labels:  attributes,
	}
}

// convertFromSum returns a single TimeSeries based on a Record with a Sum aggregation
func convertFromSum(edata exportData, sum aggregation.Sum) (prompb.TimeSeries, error) {
	// Get Sum value
	value, err := sum.Sum()
	if err != nil {
		return prompb.TimeSeries{}, err
	}

	// Create TimeSeries. Note that Cortex requires the name attribute to be in the format
	// "__name__". This is the case for all time series created by this exporter.
	name := sanitize(edata.Descriptor().Name())
	numberKind := edata.Descriptor().NumberKind()
	tSeries := createTimeSeries(edata, value, numberKind, attribute.String("__name__", name))

	return tSeries, nil
}

// convertFromLastValue returns a single TimeSeries based on a Record with a LastValue aggregation
func convertFromLastValue(edata exportData, lastValue aggregation.LastValue) (prompb.TimeSeries, error) {
	// Get value
	value, _, err := lastValue.LastValue()
	if err != nil {
		return prompb.TimeSeries{}, err
	}

	// Create TimeSeries
	name := sanitize(edata.Descriptor().Name())
	numberKind := edata.Descriptor().NumberKind()
	tSeries := createTimeSeries(edata, value, numberKind, attribute.String("__name__", name))

	return tSeries, nil
}

// convertFromMinMaxSumCount returns 4 TimeSeries for the min, max, sum, and count from the mmsc aggregation
func convertFromMinMaxSumCount(edata exportData, minMaxSumCount aggregation.MinMaxSumCount) ([]prompb.TimeSeries, error) {
	numberKind := edata.Descriptor().NumberKind()

	// Convert Min
	min, err := minMaxSumCount.Min()
	if err != nil {
		return nil, err
	}
	name := sanitize(edata.Descriptor().Name() + "_min")
	minTimeSeries := createTimeSeries(edata, min, numberKind, attribute.String("__name__", name))

	// Convert Max
	max, err := minMaxSumCount.Max()
	if err != nil {
		return nil, err
	}
	name = sanitize(edata.Descriptor().Name() + "_max")
	maxTimeSeries := createTimeSeries(edata, max, numberKind, attribute.String("__name__", name))

	// Convert Count
	count, err := minMaxSumCount.Count()
	if err != nil {
		return nil, err
	}
	name = sanitize(edata.Descriptor().Name() + "_count")
	countTimeSeries := createTimeSeries(edata, number.NewInt64Number(int64(count)), number.Int64Kind, attribute.String("__name__", name))

	// Return all timeSeries
	tSeries := []prompb.TimeSeries{
		minTimeSeries, maxTimeSeries, countTimeSeries,
	}

	return tSeries, nil
}

// convertFromHistogram returns len(histogram.Buckets) timeseries for a histogram aggregation
func convertFromHistogram(edata exportData, histogram aggregation.Histogram) ([]prompb.TimeSeries, error) {
	var timeSeries []prompb.TimeSeries
	metricName := sanitize(edata.Descriptor().Name())
	numberKind := edata.Descriptor().NumberKind()

	// Create Sum TimeSeries
	sum, err := histogram.Sum()
	if err != nil {
		return nil, err
	}
	sumTimeSeries := createTimeSeries(edata, sum, numberKind, attribute.String("__name__", metricName+"_sum"))
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
		totalCount += float64(buckets.Counts[i])
		counts[boundary] = totalCount

		// Add upper boundary as a attribute. e.g. {le="5"}
		boundaryStr := strconv.FormatFloat(boundary, 'f', -1, 64)

		// Create timeSeries and append
		boundaryTimeSeries := createTimeSeries(edata, number.NewFloat64Number(totalCount), number.Float64Kind, attribute.String("__name__", metricName), attribute.String("le", boundaryStr))
		timeSeries = append(timeSeries, boundaryTimeSeries)
	}

	// Include the +inf boundary in the total count
	totalCount += float64(buckets.Counts[len(buckets.Counts)-1])

	// Create a timeSeries for the +inf bucket and total count
	// These are the same and are both required by Prometheus-based backends

	upperBoundTimeSeries := createTimeSeries(edata, number.NewFloat64Number(totalCount), number.Float64Kind, attribute.String("__name__", metricName), attribute.String("le", "+inf"))

	countTimeSeries := createTimeSeries(edata, number.NewFloat64Number(totalCount), number.Float64Kind, attribute.String("__name__", metricName+"_count"))

	timeSeries = append(timeSeries, upperBoundTimeSeries)
	timeSeries = append(timeSeries, countTimeSeries)

	return timeSeries, nil
}

// createLabelSet combines attributes from a Record, resource, and extra attributes to create a
// slice of prompb.Label.
func createLabelSet(edata exportData, extraAttributes ...attribute.KeyValue) []prompb.Label {
	// Map ensure no duplicate label names.
	labelMap := map[string]prompb.Label{}

	// mergeAttributes merges Record and Resource attributes into a single set, giving precedence
	// to the record's attributes.
	mi := attribute.NewMergeIterator(edata.Labels(), edata.Resource.Set())
	for mi.Next() {
		attribute := mi.Label()
		key := string(attribute.Key)
		labelMap[key] = prompb.Label{
			Name:  sanitize(key),
			Value: attribute.Value.Emit(),
		}
	}

	// Add extra attributes created by the exporter like the metric name or attributes to
	// represent histogram buckets.
	for _, attribute := range extraAttributes {
		// Ensure attribute doesn't exist. If it does, notify user that a user created attribute
		// is being overwritten by a Prometheus reserved label (e.g. 'le' for histograms)
		key := string(attribute.Key)
		value := attribute.Value.AsString()
		_, found := labelMap[key]
		if found {
			log.Printf("Attribute %s is overwritten. Check if Prometheus reserved labels are used.\n", key)
		}
		labelMap[key] = prompb.Label{
			Name:  key,
			Value: value,
		}
	}

	// Create slice of labels from labelMap and return
	res := make([]prompb.Label, 0, len(labelMap))
	for _, lb := range labelMap {
		res = append(res, lb)
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
func (e *Exporter) buildMessage(timeseries []prompb.TimeSeries) ([]byte, error) {
	// Wrap the TimeSeries as a WriteRequest since Cortex requires it.
	writeRequest := &prompb.WriteRequest{
		Timeseries: timeseries,
	}

	// Convert the struct to a slice of bytes and then compress it.
	message := make([]byte, writeRequest.Size())
	written, err := writeRequest.MarshalToSizedBuffer(message)
	if err != nil {
		return nil, err
	}
	message = message[:written]
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
