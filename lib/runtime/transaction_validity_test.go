// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func Test_UnmarshalTransactionValidity(t *testing.T) {
	t.Parallel()

	txtValidity := transaction.Validity{Priority: 1}
	txnValidityResult := scale.NewResult(transaction.Validity{}, NewTransactionValidityError())
	err := txnValidityResult.Set(scale.OK, txtValidity)
	require.NoError(t, err)
	encResult, err := scale.Marshal(txnValidityResult)
	require.NoError(t, err)
	testCases := []struct {
		name        string
		encodedData []byte
		expErr      bool
		expErrMsg   string
		expValidity *transaction.Validity
	}{
		{
			name:        "ancient birth block",
			encodedData: []byte{1, 0, 5},
			expErrMsg:   "ancient birth block",
			expErr:      true,
		},
		{
			name:        "lookup failed",
			encodedData: []byte{1, 1, 0},
			expErrMsg:   "lookup failed",
			expErr:      true,
		},
		{
			name:        "unmarshal error",
			encodedData: []byte{1},
			expErrMsg:   "scale decoding transaction validity result: EOF",
			expErr:      true,
		},
		{
			name:        "valid case",
			encodedData: encResult,
			expValidity: &txtValidity,
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

func Test_InvalidTransactionValidity(t *testing.T) {
	transactionValidityErr := NewTransactionValidityError()
	invalidTransaction := NewInvalidTransaction()
	err := invalidTransaction.SetValue(Future{})
	require.NoError(t, err)
	err = transactionValidityErr.SetValue(invalidTransaction)
	require.NoError(t, err)

	expErrMsg := "invalid transaction"
	errMsg := transactionValidityErr.Error()
	require.Equal(t, expErrMsg, errMsg)

	val, err := transactionValidityErr.Value()
	require.NoError(t, err)
	_, isParentCorrectType := val.(InvalidTransaction)
	require.True(t, isParentCorrectType)

	invTransaction, ok := val.(InvalidTransaction)
	require.True(t, ok)
	childVal, err := invTransaction.Value()
	require.NoError(t, err)
	_, isChildCorrectType := childVal.(Future)
	require.True(t, isChildCorrectType)
}

func Test_UnknownTransactionValidity(t *testing.T) {
	transactionValidityErr := NewTransactionValidityError()
	unknownTransaction := NewUnknownTransaction()
	err := unknownTransaction.SetValue(NoUnsignedValidator{})
	require.NoError(t, err)
	err = transactionValidityErr.SetValue(unknownTransaction)
	require.NoError(t, err)

	expErrMsg := "validator not found"
	errMsg := transactionValidityErr.Error()
	require.Equal(t, expErrMsg, errMsg)

	val, err := transactionValidityErr.Value()
	require.NoError(t, err)
	_, isParentCorrectType := val.(UnknownTransaction)
	require.True(t, isParentCorrectType)

	unknownTransaction, ok := val.(UnknownTransaction)
	require.True(t, ok)
	childVal, err := unknownTransaction.Value()
	require.NoError(t, err)
	_, isChildCorrectType := childVal.(NoUnsignedValidator)
	require.True(t, isChildCorrectType)
}

func Test_UnknownTransactionValidity_EncodingAndDecoding(t *testing.T) {
	transactionValidityErr := NewTransactionValidityError()
	unknownTransaction := NewUnknownTransaction()
	err := unknownTransaction.SetValue(NoUnsignedValidator{})
	require.NoError(t, err)
	err = transactionValidityErr.SetValue(unknownTransaction)
	require.NoError(t, err)

	enc, err := scale.Marshal(transactionValidityErr)
	require.NoError(t, err)

	decodedTransactionValidityErr := NewTransactionValidityError()
	err = scale.Unmarshal(enc, &decodedTransactionValidityErr)
	require.NoError(t, err)
	require.Equal(t, transactionValidityErr, decodedTransactionValidityErr)

	enc2, err := scale.Marshal(transactionValidityErr)
	require.NoError(t, err)
	require.Equal(t, enc, enc2)
}
