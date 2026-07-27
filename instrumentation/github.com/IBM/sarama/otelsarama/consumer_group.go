// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelsarama

import (
	"github.com/IBM/sarama"
)

// WrapConsumerGroupHandler wraps a sarama ConsumerGroupHandler to add
// OpenTelemetry tracing to each claimed partition.
func WrapConsumerGroupHandler(handler sarama.ConsumerGroupHandler, opts ...Option) sarama.ConsumerGroupHandler {
	cfg := newConfig(opts...)
	return &consumerGroupHandler{
		ConsumerGroupHandler: handler,
		cfg:                  cfg,
	}
}

type consumerGroupHandler struct {
	sarama.ConsumerGroupHandler
	cfg *config
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	wrapped := &consumerGroupClaim{
		ConsumerGroupClaim: claim,
		dispatcher:         newConsumerMessagesDispatcherWrapper(claim, h.cfg),
	}
	go wrapped.dispatcher.Run()
	return h.ConsumerGroupHandler.ConsumeClaim(session, wrapped)
}

type consumerGroupClaim struct {
	sarama.ConsumerGroupClaim
	dispatcher *consumerMessagesDispatcherWrapper
}

func (c *consumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return c.dispatcher.Messages()
}
