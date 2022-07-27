// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package transaction_validity

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInvalidTransactionValidity(t *testing.T) {
	transactionValidityErr := NewTransactionValidityError()
	invalidTransaction := NewInvalidTransaction()
	err := invalidTransaction.Set(Future{})
	require.NoError(t, err)
	err = transactionValidityErr.Set(invalidTransaction)
	require.NoError(t, err)

	val := transactionValidityErr.Value()
	isParentCorrectType := false
	switch val.(type) {
	case InvalidTransaction:
		isParentCorrectType = true
	}
	require.True(t, isParentCorrectType)

	transaction := val.(InvalidTransaction)
	childVal := transaction.Value()
	isChildCorrectType := false
	switch childVal.(type) {
	case Future:
		isChildCorrectType = true
	}
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
	isParentCorrectType := false
	switch val.(type) {
	case UnknownTransaction:
		isParentCorrectType = true
	}
	require.True(t, isParentCorrectType)

	transaction := val.(UnknownTransaction)
	childVal := transaction.Value()
	isChildCorrectType := false
	switch childVal.(type) {
	case NoUnsignedValidator:
		isChildCorrectType = true
	}
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
