// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package transactionValidity

import (
	"errors"

	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

/// Information on a transaction's validity and, if valid, on how it relates to other transactions.
//pub type TransactionValidity = Result<ValidTransaction, TransactionValidityError>;

// TransactionValidityError Information on a transaction's validity and, if valid,
// on how it relates to other transactions. It is a result of the form:
// Result<ValidTransaction, TransactionValidityError>
// nolint
type TransactionValidityError scale.VaryingDataType

var (
	errInvalidType     = errors.New("invalid validity type")
	errInvalidResult   = errors.New("invalid error value")
	errInvalidTypeCast = errors.New("invalid type cast")
)

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (tve *TransactionValidityError) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*tve)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*tve = TransactionValidityError(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (tve *TransactionValidityError) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*tve)
	return vdt.Value()
}

// NewTransactionValidityError is constructor for TransactionValidityError
func NewTransactionValidityError() TransactionValidityError {
	vdt, err := scale.NewVaryingDataType(NewInvalidTransaction(), NewUnknownTransaction())
	if err != nil {
		panic(err)
	}
	return TransactionValidityError(vdt)
}

// UnmarshalTransactionValidity Takes the result of the validateTransaction runtime call and unmarshalls it
// TODO use custom result type here
func UnmarshalTransactionValidity(res []byte) (*transaction.Validity, *TransactionValidityError, error) {
	validTxn := transaction.Validity{}
	txnValidityErrResult := NewTransactionValidityError()
	txnValidityResult := scale.NewResult(validTxn, txnValidityErrResult)
	err := scale.Unmarshal(res, &txnValidityResult)
	if err != nil {
		return nil, nil, err
	}
	txnValidityRes, err := txnValidityResult.Unwrap()
	if err != nil {
		switch errType := err.(type) {
		case scale.WrappedErr:
			txnValidityErr, ok := errType.Err.(TransactionValidityError)
			if !ok {
				return nil, nil, errInvalidTypeCast
			}
			return nil, &txnValidityErr, nil
		default:
			return nil, nil, errInvalidResult
		}
	} else {
		switch validity := txnValidityRes.(type) {
		case transaction.Validity:
			return &validity, nil, nil
		default:
			return nil, nil, errInvalidType
		}
	}
}
