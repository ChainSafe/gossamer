// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_interfaceToTelemetryEndpoint(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		endpoints []interface{}
		expected  []*TelemetryEndpoint
	}{
		"sub element not a slice": {
			endpoints: []interface{}{
				struct{}{},
			},
		},
		"wrong interface type": {
			endpoints: []interface{}{
				[]interface{}{"string"},
			},
		},
		"wrong interface field length": {
			endpoints: []interface{}{
				[]interface{}{"wss://telemetry.polkadot.io/submit/"},
			},
		},
		"wrong interface field position": {
			endpoints: []interface{}{
				[]interface{}{float64(0), "wss://telemetry.polkadot.io/submit/"},
			},
		},
		"interface field 1 wrong type": {
			endpoints: []interface{}{
				[]interface{}{"wss://telemetry.polkadot.io/submit/", "1"},
			},
		},
		"success": {
			endpoints: []interface{}{
				[]interface{}{"wss://telemetry.polkadot.io/submit/", float64(1)},
			},
			expected: []*TelemetryEndpoint{{
				Endpoint:  "wss://telemetry.polkadot.io/submit/",
				Verbosity: 1,
			}},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			telemetryEndpoints := interfaceToTelemetryEndpoint(testCase.endpoints)
			require.Equal(t, testCase.expected, telemetryEndpoints)
		})
	}
}
