// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var errInvalidTypeCast = errors.New("invalid type cast")

// TransactionValidityError Information on a transaction's validity and, if valid,
// on how it relates to other transactions. It is a result of the form:
// Result<transaction.Validity, TransactionValidityError>
type TransactionValidityError struct {
	inner any
}

type TransactionValidityErrorValues interface {
	InvalidTransaction | UnknownTransaction
}

func setMyVaryingDataType[Value TransactionValidityErrorValues](mvdt *TransactionValidityError, value Value) {
	mvdt.inner = value
}

func (mvdt *TransactionValidityError) SetValue(value any) (err error) {
	switch value := value.(type) {
	case InvalidTransaction:
		setMyVaryingDataType(mvdt, value)
		return
	case UnknownTransaction:
		setMyVaryingDataType(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt TransactionValidityError) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case InvalidTransaction:
		return 0, mvdt.inner, nil
	case UnknownTransaction:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt TransactionValidityError) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt TransactionValidityError) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return InvalidTransaction{}, nil
	case 1:
		return UnknownTransaction{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// Error will return the error underlying TransactionValidityError
func (tve TransactionValidityError) Error() string { //skipcq: GO-W1029
	value, err := tve.Value()
	if err != nil {
		return fmt.Sprintf("getting transaction validity error value: %s", err)
	}
	err, ok := value.(error)
	if !ok {
		panic(fmt.Sprintf("unexpected value: %T %v", err, err))
	}
	return err.Error()
}

// NewTransactionValidityError is constructor for TransactionValidityError
func NewTransactionValidityError() *TransactionValidityError {
	return &TransactionValidityError{}
}

// UnmarshalTransactionValidity takes the result of the validateTransaction runtime call and unmarshalls it
// TODO use custom result issue #2780
func UnmarshalTransactionValidity(res []byte) (*transaction.Validity, error) {
	validTxn := transaction.Validity{}
	txnValidityErrResult := NewTransactionValidityError()
	txnValidityResult := scale.NewResult(validTxn, *txnValidityErrResult)
	err := scale.Unmarshal(res, &txnValidityResult)
	if err != nil {
		return nil, fmt.Errorf("scale decoding transaction validity result: %w", err)
	}
	txnValidityRes, err := txnValidityResult.Unwrap()
	if err != nil {
		scaleWrappedErr, ok := err.(scale.WrappedErr)
		if !ok {
			panic(fmt.Sprintf("unwrapping transaction validity result: %s", err))
		}
		txnValidityErr, ok := scaleWrappedErr.Err.(TransactionValidityError)
		if !ok {
			panic(fmt.Sprintf("%s: %T", errInvalidTypeCast, scaleWrappedErr.Err))
		}

		txnValidityErrValue, err := txnValidityErr.Value()
		if err != nil {
			return nil, fmt.Errorf("getting transaction validity error value: %w", err)
		}
		switch val := txnValidityErrValue.(type) {
		// TODO use custom result issue #2780
		case InvalidTransaction:
			return nil, val
		case UnknownTransaction:
			return nil, val
		default:
			panic(fmt.Sprintf("unsupported transaction validity error: %T", txnValidityErrValue))
		}
	}
	validity, ok := txnValidityRes.(transaction.Validity)
	if !ok {
		return nil, fmt.Errorf("%w", errors.New("invalid validity type"))
	}
	return &validity, nil
}
