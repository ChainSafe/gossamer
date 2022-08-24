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
		expErr      error
		expValidity *transaction.Validity
	}{
		{
			name:   "ancient birth block",
			test:   []byte{1, 0, 5},
			expErr: errAncientBirthBlock,
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			validity, err := UnmarshalTransactionValidity(c.test)
			if c.expErr == nil {
				require.NoError(t, err)
			}

			var valErr string
			if err != nil {
				valErr = err.Error()
			}
			require.Equal(t, c.expErr.Error(), valErr)
			require.Equal(t, c.expValidity, validity)
		})
	}
}
