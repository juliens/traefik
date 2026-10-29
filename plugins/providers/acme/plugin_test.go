package acme

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetEffectiveConfiguration(t *testing.T) {
	testCases := []struct {
		desc     string
		config   *Configuration
		expected *Configuration
	}{
		{
			desc:   "default CA server",
			config: &Configuration{DNSChallenge: &DNSChallenge{Provider: "bar"}},
			expected: &Configuration{
				CAServer:     defaultCAServer,
				DNSChallenge: &DNSChallenge{Provider: "bar"},
			},
		},
		{
			desc:   "deprecated DelayBeforeCheck",
			config: &Configuration{DNSChallenge: &DNSChallenge{Provider: "bar", DelayBeforeCheck: 123}},
			expected: &Configuration{
				CAServer: defaultCAServer,
				DNSChallenge: &DNSChallenge{
					Provider:         "bar",
					DelayBeforeCheck: 123,
					Propagation:      Propagation{DelayBeforeChecks: 123},
				},
			},
		},
		{
			desc:   "deprecated DisablePropagationCheck",
			config: &Configuration{DNSChallenge: &DNSChallenge{Provider: "bar", DisablePropagationCheck: true}},
			expected: &Configuration{
				CAServer: defaultCAServer,
				DNSChallenge: &DNSChallenge{
					Provider:                "bar",
					DisablePropagationCheck: true,
					Propagation:             Propagation{DisableChecks: true},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			setEffectiveConfiguration(test.config)

			assert.Equal(t, test.expected, test.config)
		})
	}
}
