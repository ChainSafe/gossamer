package transaction_validity

import "github.com/ChainSafe/gossamer/pkg/scale"

/// Information on a transaction's validity and, if valid, on how it relates to other transactions.
//pub type TransactionValidity = Result<ValidTransaction, TransactionValidityError>;

type TransactionValidityError scale.VaryingDataType

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
