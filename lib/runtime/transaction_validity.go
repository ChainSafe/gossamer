// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// TransactionValidityError Information on a transaction's validity and, if valid,
// on how it relates to other transactions. It is a result of the form:
// Result<ValidTransaction, TransactionValidityError>
type TransactionValidityError scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (tve *TransactionValidityError) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*tve)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*tve = TransactionValidityError(vdt)
	return nil
}

// Value will return the value from the underlying VaryingDataType
func (tve *TransactionValidityError) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*tve)
	return vdt.Value()
}

// Error will return the error underlying TransactionValidityError
func (tve TransactionValidityError) Error() string {
	invalidTxn, ok := tve.Value().(InvalidTransaction)
	if !ok {
		unknownTxn, ok2 := tve.Value().(UnknownTransaction)
		if !ok2 {
			return "invalid type cast"
		}
		return unknownTxn.Error()
	}
	return invalidTxn.Error()
}

// NewTransactionValidityError is constructor for TransactionValidityError
func NewTransactionValidityError() *TransactionValidityError {
	vdt, err := scale.NewVaryingDataType(NewInvalidTransaction(), NewUnknownTransaction())
	if err != nil {
		panic(err)
	}
	tve := TransactionValidityError(vdt)
	return &tve
}

var (
	ErrInvalidTxn = errors.New("invalid transaction")
	ErrUnknownTxn = errors.New("unknown transaction")
)

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
		if ok {
			txnValidityErr, ok := scaleWrappedErr.Err.(TransactionValidityError)
			if !ok {
				return nil, fmt.Errorf("%w: %T", errors.New("invalid type cast"), scaleWrappedErr.Err)
			}

			switch txnValidityErr.Value().(type) {
			// TODO use custom result issue #2780
			case InvalidTransaction:
				return nil, fmt.Errorf("%w: %s", ErrInvalidTxn, txnValidityErr.Error())
			case UnknownTransaction:
				return nil, fmt.Errorf("%w: %s", ErrUnknownTxn, txnValidityErr.Error())
			default:
				panic(fmt.Sprintf("unsupported transaction validity error: %T", txnValidityErr.Value()))
			}
		}
		return nil, fmt.Errorf("%w: %T", errors.New("invalid error value"), err)
	}
	validity, ok := txnValidityRes.(transaction.Validity)
	if !ok {
		return nil, fmt.Errorf("%w", errors.New("invalid validity type"))
	}
	return &validity, nil
}
