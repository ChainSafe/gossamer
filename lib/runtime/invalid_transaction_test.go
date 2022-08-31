// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/transaction"

	"github.com/stretchr/testify/require"
)

func TestInvalidTransactionErrors(t *testing.T) {
	testCases := []struct {
		name        string
		test        []byte
		expErr      bool
		expErrMsg   string
		expValidity *transaction.Validity
	}{
		{
			name:      "ancient birth block",
			test:      []byte{1, 0, 5},
			expErrMsg: "ancient birth block",
			expErr:    true,
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			validity, err := UnmarshalTransactionValidity(c.test)
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
