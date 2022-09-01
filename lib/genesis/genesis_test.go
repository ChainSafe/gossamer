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
			[]interface{}{"wss://telemetry.polkadot.io/submit/"},
			nil,
		},
		{
			"Test with interface field 0 wrong type",
			[]interface{}{float32(0), "wss://telemetry.polkadot.io/submit/"},
			nil,
		},
		{
			"Test with interface field 1 wrong type",
			[]interface{}{"wss://telemetry.polkadot.io/submit/", "1"},
			nil,
		},
		{
			"Test with correctly formed values",
			[]interface{}{"wss://telemetry.polkadot.io/submit/", float64(1)},
			[]*TelemetryEndpoint{{
				Endpoint:  "wss://telemetry.polkadot.io/submit/",
				Verbosity: 1,
			}},
		},
	}

	for _, test := range testcases {
		res := interfaceToTelemetryEndpoint(test.values)
		require.Equal(t, test.expected, res)
	}
}
