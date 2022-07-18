package transaction_validity

import (
	"errors"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

/// Information on a transaction's validity and, if valid, on how it relates to other transactions.
//pub type TransactionValidity = Result<ValidTransaction, TransactionValidityError>;

type TransactionValidityError scale.VaryingDataType

var (
	errInvalidType   = errors.New("invalid validity type")
	errInvalidResult = errors.New("invalid error value")
)

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (tve *TransactionValidityError) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*tve)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	// store original ParentVDT with VaryingDataType that has been set
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
	// use standard VaryingDataType constructor to construct a VaryingDataType
	vdt, err := scale.NewVaryingDataType(NewInvalidTransaction(), NewUnknownTransaction())
	if err != nil {
		panic(err)
	}
	// cast to ParentVDT
	return TransactionValidityError(vdt)
}

func DetermineValidity(res []byte) (scale.Result, error) {
	validTxn := transaction.Validity{}
	txnValidityErrResult := NewTransactionValidityError()
	txnValidityResult := scale.NewResult(validTxn, txnValidityErrResult)
	err := scale.Unmarshal(res, &txnValidityResult)
	if err != nil {
		return scale.Result{}, err
	}
	return txnValidityResult, nil
}

// TODO have this be a custom result type
func DecodeValidityError(txnValidityResult scale.Result) (*transaction.Validity, error) {
	txnValidityRes, err := txnValidityResult.Unwrap()
	if err != nil {
		switch errType := err.(type) {
		case scale.WrappedErr:
			txnValidityRes := errType.Err.(TransactionValidityError)
			switch val := txnValidityRes.Value().(type) {
			case InvalidTransaction:
				return nil, val.DetermineErrType()
			case UnknownTransaction:
				return nil, val.DetermineErrType()
			}
			return nil, errInvalidType
		default:
			return nil, errInvalidResult
		}
	} else {
		switch validity := txnValidityRes.(type) {
		case transaction.Validity:
			return &validity, nil
		default:
			return nil, errInvalidType
		}
	}
}
