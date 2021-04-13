// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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
