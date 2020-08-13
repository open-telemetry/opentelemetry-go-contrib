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

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/label"
	apimetric "go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/export/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
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
		simple.NewWithExactDistribution(),
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
func (e *Exporter) ConvertToTimeSeries(checkpointSet export.CheckpointSet) ([]*prompb.TimeSeries, error) {
	var aggError error
	var timeSeries []*prompb.TimeSeries

	// Iterate over each record in the checkpoint set and convert to TimeSeries
	aggError = checkpointSet.ForEach(e, func(record metric.Record) error {
		// Convert based on aggregation type
		agg := record.Aggregation()

		// Check if aggregation has Sum value
		if sum, ok := agg.(aggregation.Sum); ok {
			tSeries, err := convertFromSum(record, sum)
			if err != nil {
				return err
			}

			timeSeries = append(timeSeries, tSeries)
		}

		// Check if aggregation has MinMaxSumCount value
		if minMaxSumCount, ok := agg.(aggregation.MinMaxSumCount); ok {
			tSeries, err := convertFromMinMaxSumCount(record, minMaxSumCount)
			if err != nil {
				return err
			}

			timeSeries = append(timeSeries, tSeries...)
		} else if lastValue, ok := agg.(aggregation.LastValue); ok {
			tSeries, err := convertFromLastValue(record, lastValue)
			if err != nil {
				return err
			}

			timeSeries = append(timeSeries, tSeries)
		}

		return nil
	})

	// Check if error was returned in checkpointSet.ForEach()
	if aggError != nil {
		return nil, aggError
	}

	return timeSeries, nil
}

// convertFromSum returns a single TimeSeries based on a Record with a Sum aggregation
func convertFromSum(record metric.Record, sum aggregation.Sum) (*prompb.TimeSeries, error) {
	// Get Sum value
	value, err := sum.Sum()
	if err != nil {
		return nil, err
	}
	// Create sample from Sum value
	sample := prompb.Sample{
		Value:     value.CoerceToFloat64(record.Descriptor().NumberKind()),
		Timestamp: record.EndTime().Unix(), // Convert time to Unix (int64)
	}

	// Create labels, including metric name
	name := sanitize(record.Descriptor().Name())
	labels := createLabelSet(record, "__name__", name)

	// Create TimeSeries and return
	tSeries := &prompb.TimeSeries{
		Samples: []prompb.Sample{sample},
		Labels:  labels,
	}

	return tSeries, nil
}

// convertFromLastValue returns a single TimeSeries based on a Record with a LastValue aggregation
func convertFromLastValue(record metric.Record, lastValue aggregation.LastValue) (*prompb.TimeSeries, error) {
	// Get value
	value, _, err := lastValue.LastValue()
	if err != nil {
		return nil, err
	}

	// Create sample from Last value
	sample := prompb.Sample{
		Value:     value.CoerceToFloat64(record.Descriptor().NumberKind()),
		Timestamp: record.EndTime().Unix(), // Convert time to Unix (int64)
	}

	// Create labels, including metric name
	name := sanitize(record.Descriptor().Name())
	labels := createLabelSet(record, "__name__", name)

	// Create TimeSeries and return
	tSeries := &prompb.TimeSeries{
		Samples: []prompb.Sample{sample},
		Labels:  labels,
	}

	return tSeries, nil
}

// convertFromMinMaxSumCount returns 4 TimeSeries for the min, max, sum, and count from the mmsc aggregation
func convertFromMinMaxSumCount(record metric.Record, minMaxSumCount aggregation.MinMaxSumCount) ([]*prompb.TimeSeries, error) {
	// Convert Min
	min, err := minMaxSumCount.Min()
	if err != nil {
		return nil, err
	}
	minSample := prompb.Sample{
		Value:     min.CoerceToFloat64(record.Descriptor().NumberKind()),
		Timestamp: record.EndTime().Unix(), // Convert time to Unix (int64)
	}

	// Create labels, including metric name
	name := sanitize(record.Descriptor().Name() + "_min")
	labels := createLabelSet(record, "__name__", name)

	// Create TimeSeries
	minTimeSeries := &prompb.TimeSeries{
		Samples: []prompb.Sample{minSample},
		Labels:  labels,
	}

	// Convert Max
	max, err := minMaxSumCount.Max()
	if err != nil {
		return nil, err
	}
	maxSample := prompb.Sample{
		Value:     max.CoerceToFloat64(record.Descriptor().NumberKind()),
		Timestamp: record.EndTime().Unix(), // Convert time to Unix (int64)
	}

	// Create labels, including metric name
	name = sanitize(record.Descriptor().Name() + "_max")
	labels = createLabelSet(record, "__name__", name)

	// Create TimeSeries
	maxTimeSeries := &prompb.TimeSeries{
		Samples: []prompb.Sample{maxSample},
		Labels:  labels,
	}

	// Convert Count
	count, err := minMaxSumCount.Count()
	if err != nil {
		return nil, err
	}
	countSample := prompb.Sample{
		Value:     float64(count),
		Timestamp: record.EndTime().Unix(), // Convert time to Unix (int64)
	}

	// Create labels, including metric name
	name = sanitize(record.Descriptor().Name() + "_count")
	labels = createLabelSet(record, "__name__", name)

	// Create TimeSeries
	countTimeSeries := &prompb.TimeSeries{
		Samples: []prompb.Sample{countSample},
		Labels:  labels,
	}

	tSeries := []*prompb.TimeSeries{
		minTimeSeries, maxTimeSeries, countTimeSeries,
	}

	return tSeries, nil
}

// createLabelSet combines labels from a Record, resource, and extra labels to
// create a slice of prompb.Label
func createLabelSet(record metric.Record, extras ...string) []*prompb.Label {
	// Map ensure no duplicate label names
	labelMap := map[string]prompb.Label{}

	// mergeLabels merges Record and Resource labels into a single set, giving
	// precedence to the record's labels.
	mi := label.NewMergeIterator(record.Labels(), record.Resource().LabelSet())
	for mi.Next() {
		label := mi.Label()
		key := string(label.Key)
		labelMap[key] = prompb.Label{
			Name:  sanitize(key),
			Value: label.Value.Emit(),
		}
	}

	// Add extra labels created by the exporter like the metric name
	// or labels to represent histogram buckets
	for i := 0; i < len(extras); i += 2 {
		// Ensure even number of extras (key : value)
		if i+1 >= len(extras) {
			break
		}

		// Ensure label doesn't exist. If it does, notify user that a user created label
		// is being overwritten by a Prometheus reserved label (e.g. 'le' for histograms)
		_, found := labelMap[extras[i]]
		if found {
			log.Printf("Label %s is overwritten. Check if Prometheus reserved labels are used.\n", extras[i])
		}
		labelMap[extras[i]] = prompb.Label{
			Name:  sanitize(extras[i]),
			Value: extras[i+1],
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

// AddHeaders adds required headers as well as all headers in Header map to a http request.
func (e *Exporter) addHeaders(req *http.Request) {
	// Cortex expects Snappy-compressed protobuf messages. These two headers are hard-coded as they
	// should be on every request.
	req.Header.Add("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.Header.Add("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")

	// Add all user-supplied headers to the request.
	for name, field := range e.config.Headers {
		req.Header.Add(name, field)
	}
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
	e.addHeaders(req)

	return req, nil
}

// sendRequest sends an http request using the Exporter's http Client.
func (e *Exporter) sendRequest(req *http.Request) error {
	// Set a client if the user didn't provide one.
	if e.config.Client == nil {
		e.config.Client = http.DefaultClient
		e.config.Client.Timeout = e.config.RemoteTimeout
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
