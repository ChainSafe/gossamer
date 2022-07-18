// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package errors

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestApplyExtrinsicErrors(t *testing.T) {
	testValidity := transaction.Validity{
		Priority: 0x3e8,
		Requires: [][]byte{{0xb5, 0x47, 0xb1, 0x90, 0x37, 0x10, 0x7e, 0x1f, 0x79,
			0x4c, 0xa8, 0x69, 0x0, 0xa1, 0xb5, 0x98}},
		Provides: [][]byte{{0xe4, 0x80, 0x7d, 0x1b, 0x67, 0x49, 0x37, 0xbf, 0xc7,
			0x89, 0xbb, 0xdd, 0x88, 0x6a, 0xdd, 0xd6}},
		Longevity: 0x40,
		Propagate: true,
	}
	encValidity, err := scale.Marshal(testValidity)
	require.NoError(t, err)
	fmt.Println(encValidity)
	validByte := []byte{0}
	validByte = append(validByte, encValidity...)

	// test decodeing
	enc := common.MustHexToBytes("0x464c490a19b68b00000490d43593c715fdd31c61141abd04a99fd6822c855885" +
		"4ccde39a5684e7a56da27d00000000feffffffffffffff01")
	testVal := &transaction.Validity{}
	err = scale.Unmarshal(enc, testVal)
	require.NoError(t, err)

	valTest := common.MustHexToBytes("0x00464c490a19b68b00000490d43593c715fdd31c61141abd04a99fd6822c8" +
		"558854ccde39a5684e7a56da27d00000000feffffffffffffff01")
	fmt.Println(valTest)

	testCases := []struct {
		name          string
		test          []byte
		expErr        error
		expValidity   *transaction.Validity
		isValidityErr bool
	}{
		{
			name:   "lookup failed",
			test:   []byte{1, 1, 0},
			expErr: &TransactionValidityError{errLookupFailed},
		},
		{
			name:   "unexpected transaction call",
			test:   []byte{1, 0, 0},
			expErr: &TransactionValidityError{errUnexpectedTxCall},
		},
		{
			name:   "ancient birth block",
			test:   []byte{1, 0, 5},
			expErr: &TransactionValidityError{errAncientBirthBlock},
		},
		{
			name:        "valid path",
			test:        validByte,
			expValidity: &testValidity,
		},
		{
			name:        "this should decode correctly i think",
			test:        valTest,
			expValidity: testVal,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			validity, err := DecodeValidity(c.test)
			require.Equal(t, c.expErr, err)
			require.Equal(t, c.expValidity, validity)
		})
	}
}
