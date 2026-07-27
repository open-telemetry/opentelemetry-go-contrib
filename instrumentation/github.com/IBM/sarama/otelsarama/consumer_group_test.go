// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// fakeConsumerGroupSession implements sarama.ConsumerGroupSession for tests.
type fakeConsumerGroupSession struct{}

func (*fakeConsumerGroupSession) Claims() map[string][]int32                  { return nil }
func (*fakeConsumerGroupSession) MemberID() string                            { return "" }
func (*fakeConsumerGroupSession) GenerationID() int32                         { return 0 }
func (*fakeConsumerGroupSession) MarkOffset(string, int32, int64, string)     {}
func (*fakeConsumerGroupSession) ResetOffset(string, int32, int64, string)    {}
func (*fakeConsumerGroupSession) MarkMessage(*sarama.ConsumerMessage, string) {}
func (*fakeConsumerGroupSession) Context() context.Context                    { return context.Background() }
func (*fakeConsumerGroupSession) Commit()                                     {}

// fakeConsumerGroupClaim implements sarama.ConsumerGroupClaim for tests.
type fakeConsumerGroupClaim struct {
	messages chan *sarama.ConsumerMessage
}

func (c *fakeConsumerGroupClaim) Topic() string                            { return "test-topic" }
func (c *fakeConsumerGroupClaim) Partition() int32                         { return 0 }
func (c *fakeConsumerGroupClaim) InitialOffset() int64                     { return 0 }
func (c *fakeConsumerGroupClaim) HighWaterMarkOffset() int64               { return 0 }
func (c *fakeConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage { return c.messages }

// recordingHandler captures the claim passed to ConsumeClaim.
type recordingHandler struct {
	receivedClaim sarama.ConsumerGroupClaim
	done          chan struct{}
}

func (h *recordingHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *recordingHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *recordingHandler) ConsumeClaim(_ sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.receivedClaim = claim
	// Drain all messages so the dispatcher goroutine can exit.
	for range claim.Messages() {
	}
	close(h.done)
	return nil
}

func TestWrapConsumerGroupHandlerWrapsMessages(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	msgCh := make(chan *sarama.ConsumerMessage, 1)
	msgCh <- &sarama.ConsumerMessage{Topic: "test-topic", Partition: 0, Offset: 7}
	close(msgCh)

	claim := &fakeConsumerGroupClaim{messages: msgCh}
	handler := &recordingHandler{done: make(chan struct{})}
	wrapped := WrapConsumerGroupHandler(handler, WithTracerProvider(tp))

	err := wrapped.ConsumeClaim(&fakeConsumerGroupSession{}, claim)
	require.NoError(t, err)
	<-handler.done

	// The claim the inner handler received must be our instrumented wrapper.
	_, ok := handler.receivedClaim.(*consumerGroupClaim)
	assert.True(t, ok, "inner handler should receive a *consumerGroupClaim wrapper")

	spans := sr.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "process test-topic", span.Name())

	attrs := spanAttrs(span)
	assert.Equal(t, "kafka", attrs["messaging.system"])
	assert.Equal(t, "test-topic", attrs["messaging.destination.name"])
	assert.Equal(t, int64(7), attrs["messaging.kafka.offset"])
}
