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

var tcBalancesFields = []struct {
	name      string
	jsonValue []byte
	goValue   balancesFields
}{
	{
		name: "test1",
		jsonValue: []byte{
			91, 34, 53, 71, 114, 119, 118, 97, 69, 70, 53, 122, 88, 98, 50, 54, 70, 122, 57, 114, 99,
			81, 112, 68, 87, 83, 53, 55, 67, 116, 69, 82, 72, 112, 78, 101, 104, 88, 67, 80, 99, 78,
			111, 72, 71, 75, 117, 116, 81, 89, 34, 44, 49, 50, 51, 52, 50, 51, 52, 50, 51, 52, 93,
		},
		goValue: balancesFields{"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY", 1234234234},
	},
}

func TestBalancesFieldsMarshal(t *testing.T) {
	for _, tt := range tcBalancesFields {
		t.Run(tt.name, func(t *testing.T) {
			marshalledValue, err := tt.goValue.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, tt.jsonValue, marshalledValue)
		})
	}
}

func TestBalancesFieldsUnmarshal(t *testing.T) {
	for _, tt := range tcBalancesFields {
		t.Run(tt.name, func(t *testing.T) {
			var bfs balancesFields
			err := bfs.UnmarshalJSON(tt.jsonValue)
			require.NoError(t, err)
			require.EqualValues(t, bfs, tt.goValue)
		})
	}
}
