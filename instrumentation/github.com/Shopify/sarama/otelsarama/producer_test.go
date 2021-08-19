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

package otelsarama

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestAsyncProducer_ConcurrencyEdgeCases(t *testing.T) {
	cfg := newSaramaConfig()
	testCases := []struct {
		name             string
		newAsyncProducer func(t *testing.T) sarama.AsyncProducer
	}{
		{
			name: "original",
			newAsyncProducer: func(t *testing.T) sarama.AsyncProducer {
				return mocks.NewAsyncProducer(t, cfg)
			},
		},
		{
			name: "wrapped",
			newAsyncProducer: func(t *testing.T) sarama.AsyncProducer {
				var ap sarama.AsyncProducer = mocks.NewAsyncProducer(t, cfg)
				ap = WrapAsyncProducer(cfg, ap)
				return ap
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("closes Successes and Error after Close", func(t *testing.T) {
				timeout := time.NewTimer(time.Minute)
				defer timeout.Stop()
				p := tc.newAsyncProducer(t)

				p.Close()

				select {
				case <-timeout.C:
					t.Error("timeout - Successes channel was not closed")
				case _, ok := <-p.Successes():
					if ok {
						t.Error("message was send to Successes channel instead of being closed")
					}
				}

				select {
				case <-timeout.C:
					t.Error("timeout - Errors channel was not closed")
				case _, ok := <-p.Errors():
					if ok {
						t.Error("message was send to Errors channel instead of being closed")
					}
				}
			})

			t.Run("closes Successes and Error after AsyncClose", func(t *testing.T) {
				timeout := time.NewTimer(time.Minute)
				defer timeout.Stop()
				p := tc.newAsyncProducer(t)

				p.AsyncClose()

				select {
				case <-timeout.C:
					t.Error("timeout - Successes channel was not closed")
				case _, ok := <-p.Successes():
					if ok {
						t.Error("message was send to Successes channel instead of being closed")
					}
				}

				select {
				case <-timeout.C:
					t.Error("timeout - Errors channel was not closed")
				case _, ok := <-p.Errors():
					if ok {
						t.Error("message was send to Errors channel instead of being closed")
					}
				}
			})

			t.Run("panic when sending to Input after Close", func(t *testing.T) {
				p := tc.newAsyncProducer(t)
				p.Close()
				assert.Panics(t, func() {
					p.Input() <- &sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}
				})
			})

			t.Run("panic when sending to Input after AsyncClose", func(t *testing.T) {
				p := tc.newAsyncProducer(t)
				p.AsyncClose()
				assert.Panics(t, func() {
					p.Input() <- &sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}
				})
			})

			t.Run("panic when calling Close after AsyncClose", func(t *testing.T) {
				p := tc.newAsyncProducer(t)
				p.AsyncClose()
				assert.Panics(t, func() {
					p.Close()
				})
			})

			t.Run("panic when calling AsyncClose after Close", func(t *testing.T) {
				p := tc.newAsyncProducer(t)
				p.Close()
				assert.Panics(t, func() {
					p.AsyncClose()
				})
			})
		})
	}
}

func newSaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0
	return cfg
}

func BenchmarkWrapSyncProducer(b *testing.B) {
	// Mock provider
	provider := oteltrace.NewNoopTracerProvider()

	cfg := newSaramaConfig()
	// Mock sync producer
	mockSyncProducer := mocks.NewSyncProducer(b, cfg)

	// Wrap sync producer
	syncProducer := WrapSyncProducer(cfg, mockSyncProducer, WithTracerProvider(provider))
	message := sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockSyncProducer.ExpectSendMessageAndSucceed()
		_, _, err := syncProducer.SendMessage(&message)
		assert.NoError(b, err)
	}
}

func BenchmarkMockSyncProducer(b *testing.B) {
	cfg := newSaramaConfig()
	// Mock sync producer
	mockSyncProducer := mocks.NewSyncProducer(b, cfg)

	// Wrap sync producer
	syncProducer := mockSyncProducer
	message := sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockSyncProducer.ExpectSendMessageAndSucceed()
		_, _, err := syncProducer.SendMessage(&message)
		assert.NoError(b, err)
	}
}

func BenchmarkWrapAsyncProducer(b *testing.B) {
	// Mock provider
	provider := oteltrace.NewNoopTracerProvider()

	cfg := newSaramaConfig()
	cfg.Producer.Return.Successes = true
	mockAsyncProducer := mocks.NewAsyncProducer(b, cfg)

	// Wrap sync producer
	asyncProducer := WrapAsyncProducer(cfg, mockAsyncProducer, WithTracerProvider(provider))
	message := sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockAsyncProducer.ExpectInputAndSucceed()
		asyncProducer.Input() <- &message
		<-asyncProducer.Successes()
	}
}

func BenchmarkMockAsyncProducer(b *testing.B) {
	cfg := newSaramaConfig()
	cfg.Producer.Return.Successes = true
	mockAsyncProducer := mocks.NewAsyncProducer(b, cfg)

	// Wrap sync producer
	asyncProducer := mockAsyncProducer
	message := sarama.ProducerMessage{Key: sarama.StringEncoder("foo")}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockAsyncProducer.ExpectInputAndSucceed()
		mockAsyncProducer.Input() <- &message
		<-asyncProducer.Successes()
	}
}
