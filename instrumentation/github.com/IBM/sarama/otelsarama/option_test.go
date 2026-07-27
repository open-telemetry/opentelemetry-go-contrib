// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClient is a minimal sarama.Client stub that returns a controlled broker
// list for testing fetchClusterID without a real Kafka cluster.
type fakeClient struct {
	brokers []*sarama.Broker
}

func (*fakeClient) Config() *sarama.Config                                      { return sarama.NewConfig() }
func (*fakeClient) Controller() (*sarama.Broker, error)                         { return nil, nil }
func (*fakeClient) RefreshController() (*sarama.Broker, error)                  { return nil, nil }
func (f *fakeClient) Brokers() []*sarama.Broker                                 { return f.brokers }
func (*fakeClient) Broker(int32) (*sarama.Broker, error)                        { return nil, nil }
func (*fakeClient) Topics() ([]string, error)                                   { return nil, nil }
func (*fakeClient) Partitions(string) ([]int32, error)                          { return nil, nil }
func (*fakeClient) WritablePartitions(string) ([]int32, error)                  { return nil, nil }
func (*fakeClient) Leader(string, int32) (*sarama.Broker, error)                { return nil, nil }
func (*fakeClient) LeaderAndEpoch(string, int32) (*sarama.Broker, int32, error) { return nil, 0, nil }
func (*fakeClient) Replicas(string, int32) ([]int32, error)                     { return nil, nil }
func (*fakeClient) InSyncReplicas(string, int32) ([]int32, error)               { return nil, nil }
func (*fakeClient) OfflineReplicas(string, int32) ([]int32, error)              { return nil, nil }
func (*fakeClient) RefreshBrokers([]string) error                               { return nil }
func (*fakeClient) RefreshMetadata(...string) error                             { return nil }
func (*fakeClient) GetOffset(string, int32, int64) (int64, error)               { return 0, nil }
func (*fakeClient) Coordinator(string) (*sarama.Broker, error)                  { return nil, nil }
func (*fakeClient) RefreshCoordinator(string) error                             { return nil }
func (*fakeClient) TransactionCoordinator(string) (*sarama.Broker, error)       { return nil, nil }
func (*fakeClient) RefreshTransactionCoordinator(string) error                  { return nil }
func (*fakeClient) InitProducerID() (*sarama.InitProducerIDResponse, error)     { return nil, nil }
func (*fakeClient) LeastLoadedBroker() *sarama.Broker                           { return nil }
func (*fakeClient) Close() error                                                { return nil }
func (*fakeClient) Closed() bool                                                { return false }

func TestFetchClusterIDEmptyBrokers(t *testing.T) {
	cfg := newConfig()
	fetchClusterID(cfg, &fakeClient{brokers: nil})
	assert.Nil(t, cfg.clusterID.Load())
}

func TestFetchClusterIDSuccess(t *testing.T) {
	mockBroker := sarama.NewMockBroker(t, 1)
	defer mockBroker.Close()

	clusterID := "test-cluster"
	mockBroker.Returns(&sarama.MetadataResponse{Version: 4, ClusterID: &clusterID})

	broker := sarama.NewBroker(mockBroker.Addr())
	require.NoError(t, broker.Open(sarama.NewConfig()))
	defer broker.Close()

	cfg := newConfig()
	fetchClusterID(cfg, &fakeClient{brokers: []*sarama.Broker{broker}})

	got := cfg.clusterID.Load()
	require.NotNil(t, got)
	assert.Equal(t, "test-cluster", *got)
}

func TestWithClientSetsClusterID(t *testing.T) {
	mockBroker := sarama.NewMockBroker(t, 1)
	defer mockBroker.Close()

	clusterID := "with-client-cluster"
	mockBroker.Returns(&sarama.MetadataResponse{Version: 4, ClusterID: &clusterID})

	broker := sarama.NewBroker(mockBroker.Addr())
	require.NoError(t, broker.Open(sarama.NewConfig()))
	defer broker.Close()

	cfg := newConfig(WithClient(&fakeClient{brokers: []*sarama.Broker{broker}}))

	// fetchClusterID runs in a goroutine; poll briefly until it stores the ID.
	require.Eventually(t, func() bool {
		return cfg.clusterID.Load() != nil
	}, 100*time.Millisecond, time.Millisecond)

	assert.Equal(t, "with-client-cluster", *cfg.clusterID.Load())
}
