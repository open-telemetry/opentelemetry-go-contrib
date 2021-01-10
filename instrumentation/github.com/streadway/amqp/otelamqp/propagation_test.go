package otelamqp

import "testing"
import "github.com/stretchr/testify/assert"

func TestPropagation(t *testing.T) {
	data := map[string]interface{}{
		"a": "b",
	}
	c := amqpHeadersCarrier(data)
	c.Set("c", "d")

	assert.Equal(t, "d", c.Get("c"))
}
