// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// InvalidTransaction is a child VDT of TransactionValidityError
type InvalidTransaction struct {
	inner any
}

type InvalidTransactionValues interface {
	Call | Payment | Future | Stale | BadProof | AncientBirthBlock |
		ExhaustsResources | InvalidCustom | BadMandatory | MandatoryDispatch | BadSigner
}

func setInvalidTransaction[Value InvalidTransactionValues](it *InvalidTransaction, value Value) {
	it.inner = value
}

func (it *InvalidTransaction) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Call:
		setInvalidTransaction(it, value)
		return
	case Payment:
		setInvalidTransaction(it, value)
		return
	case Future:
		setInvalidTransaction(it, value)
		return
	case Stale:
		setInvalidTransaction(it, value)
		return
	case BadProof:
		setInvalidTransaction(it, value)
		return
	case AncientBirthBlock:
		setInvalidTransaction(it, value)
		return
	case ExhaustsResources:
		setInvalidTransaction(it, value)
		return
	case InvalidCustom:
		setInvalidTransaction(it, value)
		return
	case BadMandatory:
		setInvalidTransaction(it, value)
		return
	case MandatoryDispatch:
		setInvalidTransaction(it, value)
		return
	case BadSigner:
		setInvalidTransaction(it, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (it InvalidTransaction) IndexValue() (index uint, value any, err error) {
	switch it.inner.(type) {
	case Call:
		return 0, it.inner, nil
	case Payment:
		return 1, it.inner, nil
	case Future:
		return 2, it.inner, nil
	case Stale:
		return 3, it.inner, nil
	case BadProof:
		return 4, it.inner, nil
	case AncientBirthBlock:
		return 5, it.inner, nil
	case ExhaustsResources:
		return 6, it.inner, nil
	case InvalidCustom:
		return 7, it.inner, nil
	case BadMandatory:
		return 8, it.inner, nil
	case MandatoryDispatch:
		return 9, it.inner, nil
	case BadSigner:
		return 10, it.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (it InvalidTransaction) Value() (value any, err error) {
	_, value, err = it.IndexValue()
	return
}

func (it InvalidTransaction) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return Call{}, nil
	case 1:
		return Payment{}, nil
	case 2:
		return Future{}, nil
	case 3:
		return Stale{}, nil
	case 4:
		return BadProof{}, nil
	case 5:
		return AncientBirthBlock{}, nil
	case 6:
		return ExhaustsResources{}, nil
	case 7:
		return InvalidCustom(0), nil
	case 8:
		return BadMandatory{}, nil
	case 9:
		return MandatoryDispatch{}, nil
	case 10:
		return BadSigner{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// // Index returns the VDT index
// func (InvalidTransaction) Index() uint { //skipcq: GO-W1029
// 	return 0
// }

// func (i InvalidTransaction) String() string { return i.Error() } //skipcq: GO-W1029

// // Set will set a VaryingDataTypeValue using the underlying VaryingDataType
// func (i *InvalidTransaction) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
// 	vdt := scale.VaryingDataType(*i)
// 	err = vdt.Set(val)
// 	if err != nil {
// 		return err
// 	}
// 	*i = InvalidTransaction(vdt)
// 	return nil
// }

// // Value will return the value from the underying VaryingDataType
// func (i *InvalidTransaction) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
// 	vdt := scale.VaryingDataType(*i)
// 	return vdt.Value()
// }

// Error returns the error message associated with the InvalidTransaction
func (i InvalidTransaction) Error() string { //skipcq: GO-W1029
	value, err := i.Value()
	if err != nil {
		return fmt.Sprintf("getting invalid transaction value: %s", err)
	}
	err, ok := value.(error)
	if !ok {
		panic(fmt.Sprintf("%T does not implement the error type", value))
	}
	return err.Error()
}

// NewInvalidTransaction is constructor for InvalidTransaction
func NewInvalidTransaction() InvalidTransaction {
	return InvalidTransaction{}
}

// Call The call of the transaction is not expected
type Call struct{}

// Payment General error to do with the inability to pay some fees (e.g. account balance too low)
type Payment struct{}

// Future General error to do with the transaction not yet being valid (e.g. nonce too high)
type Future struct{}

// Stale General error to do with the transaction being outdated (e.g. nonce too low)
type Stale struct{}

// BadProof General error to do with the transaction’s proofs (e.g. signature)
type BadProof struct{}

// AncientBirthBlock The transaction birth block is ancient
type AncientBirthBlock struct{}

// ExhaustsResources The transaction would exhaust the resources of current block
type ExhaustsResources struct{}

// InvalidCustom Any other custom invalid validity that is not covered
type InvalidCustom uint8

// BadMandatory An extrinsic with a Mandatory dispatch resulted in Error
type BadMandatory struct{}

// MandatoryDispatch A transaction with a mandatory dispatch
type MandatoryDispatch struct{}

// BadSigner A transaction with a mandatory dispatch
type BadSigner struct{}
