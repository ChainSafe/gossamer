// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package errors

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func Test_ErrorsAs_Function(t *testing.T) {
	transactionValidityErr := NewTransactionValidityError()
	unknownTransaction := NewUnknownTransaction()
	err := unknownTransaction.Set(NoUnsignedValidator{})
	require.NoError(t, err)
	err = transactionValidityErr.Set(unknownTransaction)
	require.NoError(t, err)

	var txnValErr *TransactionValidityError
	isTxnValErr := errors.As(&transactionValidityErr, &txnValErr)
	require.True(t, isTxnValErr)
}

func Test_InvalidTransactionValidity(t *testing.T) {
	transactionValidityErr := NewTransactionValidityError()
	invalidTransaction := NewInvalidTransaction()
	err := invalidTransaction.Set(Future{})
	require.NoError(t, err)
	err = transactionValidityErr.Set(invalidTransaction)
	require.NoError(t, err)

	val := transactionValidityErr.Value()
	_, isParentCorrectType := val.(InvalidTransaction)
	require.True(t, isParentCorrectType)

	transaction, ok := val.(InvalidTransaction)
	require.True(t, ok)
	childVal := transaction.Value()
	_, isChildCorrectType := childVal.(Future)
	require.True(t, isChildCorrectType)
}

func Test_UnknownTransactionValidity(t *testing.T) {
	transactionValidityErr := NewTransactionValidityError()
	unknownTransaction := NewUnknownTransaction()
	err := unknownTransaction.Set(NoUnsignedValidator{})
	require.NoError(t, err)
	err = transactionValidityErr.Set(unknownTransaction)
	require.NoError(t, err)

	val := transactionValidityErr.Value()
	_, isParentCorrectType := val.(UnknownTransaction)
	require.True(t, isParentCorrectType)

	transaction, ok := val.(UnknownTransaction)
	require.True(t, ok)
	childVal := transaction.Value()
	_, isChildCorrectType := childVal.(NoUnsignedValidator)
	require.True(t, isChildCorrectType)
}

func Test_UnknownTransactionValidity_EncodingAndDecoding(t *testing.T) {
	transactionValidityErr := NewTransactionValidityError()
	unknownTransaction := NewUnknownTransaction()
	err := unknownTransaction.Set(NoUnsignedValidator{})
	require.NoError(t, err)
	err = transactionValidityErr.Set(unknownTransaction)
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
