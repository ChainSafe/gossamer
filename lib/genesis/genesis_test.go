// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterfaceToTelemetryEndpoint(t *testing.T) {
	testcases := []struct {
		description string
		values      []interface{}
		expected    []*TelemetryEndpoint
	}{
		{
			"Test with wrong interface type",
			[]interface{}{"string"},
			nil,
		},
		{
			"Test with interface field len != 2",
			append(testEndpoints, []interface{}{"wss://telemetry.polkadot.io/submit/"}),
			nil,
		},
		{
			"Test with interface field 0 wrong type",
			append(testEndpoints, []interface{}{float32(0), "wss://telemetry.polkadot.io/submit/"}),
			nil,
		},
		{
			"Test with interface field 1 wrong type",
			append(testEndpoints, []interface{}{"wss://telemetry.polkadot.io/submit/", "1"}),
			nil,
		},
		{
			"Test with correctly formed values",
			append(testEndpoints, testEndpoint1),
			append([]*TelemetryEndpoint{}, &TelemetryEndpoint{
				Endpoint:  "wss://telemetry.polkadot.io/submit/",
				Verbosity: 1,
			}),
		},
	}

	for _, test := range testcases {
		res := interfaceToTelemetryEndpoint(test.values)
		require.Equal(t, test.expected, res)
	}
}
