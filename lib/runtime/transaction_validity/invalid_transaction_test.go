// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package transactionValidity

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/transaction"

	"github.com/stretchr/testify/require"
)

func TestInvalidTransactionErrors(t *testing.T) {
	testCases := []struct {
		name          string
		test          []byte
		expErr        error
		expValidity   *transaction.Validity
		isValidityErr bool
	}{
		{
			name:   "ancient birth block",
			test:   []byte{1, 0, 5},
			expErr: errAncientBirthBlock,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			validity, validityErr, err := UnmarshalTransactionValidity(c.test)
			require.NoError(t, err)

			var valErr string
			if validityErr != nil {
				valErr = validityErr.Error()
			}
			require.Equal(t, c.expErr.Error(), valErr)
			require.Equal(t, c.expValidity, validity)
		})
	}
}
