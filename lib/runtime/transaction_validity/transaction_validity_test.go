// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package transactionValidity

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestInvalidTransactionValidity(t *testing.T) {
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

func TestUnknownTransactionValidity(t *testing.T) {
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

func TestUnknownTransactionValidityEncodingAndDecoding(t *testing.T) {
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
