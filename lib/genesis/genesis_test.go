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
		"sub_element_not_a_slice": {
			endpoints: []interface{}{
				struct{}{},
			},
		},
		"wrong_interface_type": {
			endpoints: []interface{}{
				[]interface{}{"string"},
			},
		},
		"wrong_interface_field_length": {
			endpoints: []interface{}{
				[]interface{}{"wss://telemetry.polkadot.io/submit/"},
			},
		},
		"wrong_interface_field_position": {
			endpoints: []interface{}{
				[]interface{}{float64(0), "wss://telemetry.polkadot.io/submit/"},
			},
		},
		"interface_field_1_wrong_type": {
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
