// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package transaction_validity

import (
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUnknownTransactionErrors(t *testing.T) {
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
			expErr: errLookupFailed,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			validity, validityErr, err := UnmarshalTransactionValidity(c.test)
			require.NoError(t, err)

			var valErr error
			if validityErr != nil {
				switch err := validityErr.Value().(type) {
				// TODO with custom result type have Error() func for txnValidityErr
				case InvalidTransaction:
					valErr = err.Error()
				case UnknownTransaction:
					valErr = err.Error()
				}
			}
			require.Equal(t, c.expErr, valErr)
			require.Equal(t, c.expValidity, validity)
		})
	}
}
