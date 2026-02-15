package otelaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.opentelemetry.io/otel/propagation"
)

// SQSMessageAttributeCarrier implements propagation.TextMapCarrier for SQS message attributes
type SQSMessageAttributeCarrier struct {
	Attributes map[string]types.MessageAttributeValue
}

func (c *SQSMessageAttributeCarrier) Get(key string) string {
	if attr, exists := c.Attributes[key]; exists && attr.StringValue != nil {
		return *attr.StringValue
	}
	return ""
}

func (c *SQSMessageAttributeCarrier) Set(key, value string) {
	if c.Attributes == nil {
		c.Attributes = make(map[string]types.MessageAttributeValue)
	}
	c.Attributes[key] = types.MessageAttributeValue{
		StringValue: aws.String(value),
		DataType:    aws.String("String"),
	}
}

func (c *SQSMessageAttributeCarrier) Keys() []string {
	keys := make([]string, 0, len(c.Attributes))
	for key := range c.Attributes {
		keys = append(keys, key)
	}
	return keys
}

// SQSMessageAttributePropagator implements propagation.TextMapPropagator for SQS message attributes
type SQSMessageAttributePropagator struct {
	propagator propagation.TextMapPropagator
}

func NewSQSMessageAttributePropagator() *SQSMessageAttributePropagator {
	return &SQSMessageAttributePropagator{
		propagator: propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	}
}

func (p *SQSMessageAttributePropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	if sqsCarrier, ok := carrier.(*SQSMessageAttributeCarrier); ok {
		p.propagator.Inject(ctx, sqsCarrier)
	}
}

func (p *SQSMessageAttributePropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	if sqsCarrier, ok := carrier.(*SQSMessageAttributeCarrier); ok {
		return p.propagator.Extract(ctx, sqsCarrier)
	}
	return ctx
}

func (p *SQSMessageAttributePropagator) Fields() []string {
	return p.propagator.Fields()
}

// Ensure SQSMessageAttributePropagator implements propagation.TextMapPropagator
// SQSMessageAttributeCarrier implements propagation.TextMapCarrier
var (
	_ propagation.TextMapPropagator = &SQSMessageAttributePropagator{}
	_ propagation.TextMapCarrier    = &SQSMessageAttributeCarrier{}
)
