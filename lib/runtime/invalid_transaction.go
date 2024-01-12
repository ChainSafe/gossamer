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

func (c Call) String() string { return c.Error() }

// Error returns the error message associated with the Call
func (Call) Error() string {
	return "call of the transaction is not expected"
}

// Payment General error to do with the inability to pay some fees (e.g. account balance too low)
type Payment struct{}

func (p Payment) String() string { return p.Error() }

// Error returns the error message associated with the Payment
func (Payment) Error() string {
	return "invalid payment"
}

// Future General error to do with the transaction not yet being valid (e.g. nonce too high)
type Future struct{}

func (f Future) String() string { return f.Error() }

// Error returns the error message associated with the Future
func (Future) Error() string {
	return "invalid transaction"
}

// Stale General error to do with the transaction being outdated (e.g. nonce too low)
type Stale struct{}

func (s Stale) String() string { return s.Error() }

// Error returns the error message associated with the Stale
func (Stale) Error() string {
	return "outdated transaction"
}

// BadProof General error to do with the transaction’s proofs (e.g. signature)
type BadProof struct{}

func (b BadProof) String() string { return b.Error() }

// Error returns the error message associated with the BadProof
func (BadProof) Error() string {
	return "bad proof"
}

// AncientBirthBlock The transaction birth block is ancient
type AncientBirthBlock struct{}

func (a AncientBirthBlock) String() string { return a.Error() }

// Error returns the error message associated with the AncientBirthBlock
func (AncientBirthBlock) Error() string {
	return "ancient birth block"
}

// ExhaustsResources The transaction would exhaust the resources of current block
type ExhaustsResources struct{}

func (e ExhaustsResources) String() string { return e.Error() }

// Error returns the error message associated with the ExhaustsResources
func (ExhaustsResources) Error() string {
	return "exhausts resources"
}

// InvalidCustom Any other custom invalid validity that is not covered
type InvalidCustom uint8

func (i InvalidCustom) String() string { return i.Error() }

// Error returns the error message associated with the InvalidCustom
func (i InvalidCustom) Error() string {
	return newUnknownError(i).Error()
}

// BadMandatory An extrinsic with a Mandatory dispatch resulted in Error
type BadMandatory struct{}

func (b BadMandatory) String() string { return b.Error() }

// Error returns the error message associated with the BadMandatory
func (BadMandatory) Error() string {
	return "mandatory dispatch error"
}

// MandatoryDispatch A transaction with a mandatory dispatch
type MandatoryDispatch struct{}

func (m MandatoryDispatch) String() string { return m.Error() }

// Error returns the error message associated with the MandatoryDispatch
func (MandatoryDispatch) Error() string {
	return "invalid mandatory dispatch"
}

// BadSigner A transaction with a mandatory dispatch
type BadSigner struct{}

func (b BadSigner) String() string { return b.Error() }

// Error returns the error message associated with the MandatoryDispatch
func (BadSigner) Error() string {
	return "invalid signing address"
}
