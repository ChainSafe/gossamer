// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

var tcNextKey = []struct {
	name      string
	jsonValue []byte
	goValue   nextKey
}{
	{
		name:      "test1",
		jsonValue: common.MustHexToBytes("0x5b2235474e4a715450794e71414e426b55564d4e314c50507278586e466f7557586f6532774e536d6d456f4c637478695a59222c2235474e4a715450794e71414e426b55564d4e314c50507278586e466f7557586f6532774e536d6d456f4c637478695a59222c7b226772616e647061223a22354641396e51445667323637444564386d315a7970584c426e764e37534678597756376e6471535947694e3954547075222c2262616265223a223547727776614546357a58623236467a397263517044575335374374455248704e6568584350634e6f48474b75745159222c22696d5f6f6e6c696e65223a223547727776614546357a58623236467a397263517044575335374374455248704e6568584350634e6f48474b75745159222c22706172615f76616c696461746f72223a223547727776614546357a58623236467a397263517044575335374374455248704e6568584350634e6f48474b75745159222c22706172615f61737369676e6d656e74223a223547727776614546357a58623236467a397263517044575335374374455248704e6568584350634e6f48474b75745159222c22617574686f726974795f646973636f76657279223a223547727776614546357a58623236467a397263517044575335374374455248704e6568584350634e6f48474b75745159227d5d"), //nolint:lll
		goValue: nextKey{
			"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
			"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
			keyOwner{
				Grandpa:            "5FA9nQDVg267DEd8m1ZypXLBnvN7SFxYwV7ndqSYGiN9TTpu",
				Babe:               "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
				ImOnline:           "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
				ParaValidator:      "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
				ParaAssignment:     "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
				AuthorityDiscovery: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
			},
		},
	},
}

func TestNextKeyMarshal(t *testing.T) {
	for _, tt := range tcNextKey {
		t.Run(tt.name, func(t *testing.T) {
			marshalledValue, err := tt.goValue.MarshalJSON()
			require.NoError(t, err, "couldn't marshal nextKey")
			require.EqualValues(t, tt.jsonValue, marshalledValue, "nextKey doesn't match")
		})
	}
}

func TestNextKeyUnmarshal(t *testing.T) {
	for _, tt := range tcNextKey {
		t.Run(tt.name, func(t *testing.T) {
			var nk nextKey
			err := nk.UnmarshalJSON(tt.jsonValue)
			require.NoError(t, err, "couldn't unmarshal nextKey")
			require.EqualValues(t, tt.goValue, nk, "nextKey doesn't match")
		})
	}
}
