// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package errors

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/transaction"

	"github.com/stretchr/testify/require"
)

func Test_UnknownTransaction_Errors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		encodedData []byte
		expErr      bool
		expErrMsg   string
		expValidity *transaction.Validity
	}{
		{
			name:        "lookup failed",
			encodedData: []byte{1, 1, 0},
			expErrMsg:   fmt.Errorf("%w: %s", ErrUnknownTxn, "lookup failed").Error(),
			expErr:      true,
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			validity, err := UnmarshalTransactionValidity(c.encodedData)
			if !c.expErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.EqualError(t, err, c.expErrMsg)
			}

			require.Equal(t, c.expValidity, validity)
		})
	}
}
