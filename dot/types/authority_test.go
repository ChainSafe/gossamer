// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var tcAuthorityAsAddress = []struct {
	name      string
	jsonValue []byte
	goValue   AuthorityAsAddress
}{
	{
		name: "test1",
		jsonValue: []byte{
			91, 34, 53, 71, 114, 119, 118, 97, 69, 70, 53, 122, 88, 98, 50, 54, 70, 122,
			57, 114, 99, 81, 112, 68, 87, 83, 53, 55, 67, 116, 69, 82, 72, 112, 78, 101,
			104, 88, 67, 80, 99, 78, 111, 72, 71, 75, 117, 116, 81, 89, 34, 44, 49, 93,
		},
		goValue: AuthorityAsAddress{Address: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY", Weight: 1},
	},
}

func TestAuthorityAsAddressMarshal(t *testing.T) {
	for _, tt := range tcAuthorityAsAddress {
		t.Run(tt.name, func(t *testing.T) {
			marshalledValue, err := tt.goValue.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, tt.jsonValue, marshalledValue)
		})
	}

}

func TestAuthorityAsAddressUnmarshal(t *testing.T) {
	for _, tt := range tcAuthorityAsAddress {
		t.Run(tt.name, func(t *testing.T) {
			var authorityAsAddress AuthorityAsAddress
			err := authorityAsAddress.UnmarshalJSON(tt.jsonValue)
			require.NoError(t, err)
			require.EqualValues(t, tt.goValue, authorityAsAddress)
		})
	}

}
