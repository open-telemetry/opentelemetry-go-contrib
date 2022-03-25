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
	"github.com/Shopify/sarama"
)

type consumerGroupHandler struct {
	sarama.ConsumerGroupHandler

	cfg config
}

// ConsumeClaim wraps the session and claim to add instruments for messages.
// It implements parts of `ConsumerGroupHandler`.
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Wrap claim
	dispatcher := newConsumerMessagesDispatcherWrapper(claim, h.cfg)
	go dispatcher.Run()
	claim = &consumerGroupClaim{
		ConsumerGroupClaim: claim,
		dispatcher:         dispatcher,
	}

	return h.ConsumerGroupHandler.ConsumeClaim(session, claim)
}

// WrapConsumerGroupHandler wraps a sarama.ConsumerGroupHandler causing each received
// message to be traced.
func WrapConsumerGroupHandler(handler sarama.ConsumerGroupHandler, opts ...Option) sarama.ConsumerGroupHandler {
	cfg := newConfig(opts...)

	return &consumerGroupHandler{
		ConsumerGroupHandler: handler,
		cfg:                  cfg,
	}
}

type consumerGroupClaim struct {
	sarama.ConsumerGroupClaim
	dispatcher consumerMessagesDispatcher
}

func (c *consumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return c.dispatcher.Messages()
}
