package transaction_validity

import (
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/stretchr/testify/require"
	"testing"
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
			validityResult, err := DetermineValidity(c.test)
			require.NoError(t, err)
			validity, err := DecodeValidity(validityResult)
			require.Equal(t, c.expErr, err)
			require.Equal(t, c.expValidity, validity)
		})
	}
}
