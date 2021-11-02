package jaegerremote

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	testCases := map[string]struct {
		options  []Option
		expected *config
	}{
		"default": {
			expected: &config{
				service:             "",
				endpoint:            "http://localhost:5778",
				pollingInterval:     time.Minute,
				initialSamplingRate: 0.001,
			},
		},
		"with options": {
			options: []Option{
				WithService("myService"),
				WithEndpoint("http://otel-collector:5778"),
				WithPollingInterval(5 * time.Second),
				WithInitialSamplingRate(0.5),
			},
			expected: &config{
				service:             "myService",
				endpoint:            "http://otel-collector:5778",
				pollingInterval:     5 * time.Second,
				initialSamplingRate: 0.5,
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			cfg := defaultConfig()

			for _, option := range testCase.options {
				option.apply(cfg)
			}

			assert.Equal(t, testCase.expected, cfg)
		})
	}
}
