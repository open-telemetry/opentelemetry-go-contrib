package otelamqp

// amqpHeadersCarrier satisfies with TextMapPropagator
type amqpHeadersCarrier map[string]interface{}

// Get returns the value associated with the passed key.
func (c amqpHeadersCarrier) Get(key string) string {
	for k, v := range c {
		if k== key{
			return v.(string)
		}
	}
	return ""
}

// Set stores the key-value pair.
func (c amqpHeadersCarrier) Set(key, val string) {
	c[key] = val
}