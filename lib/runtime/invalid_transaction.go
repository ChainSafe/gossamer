// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"errors"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// InvalidTransaction is child VDT of TransactionValidityError
type InvalidTransaction scale.VaryingDataType

// Index fulfils the VaryingDataTypeValue interface.  T
func (InvalidTransaction) Index() uint {
	return 0
}

var (
	errUnexpectedTxCall         = errors.New("call of the transaction is not expected")
	errInvalidPayment           = errors.New("invalid payment")
	errInvalidTransaction       = errors.New("invalid transaction")
	errOutdatedTransaction      = errors.New("outdated transaction")
	errBadProof                 = errors.New("bad proof")
	errAncientBirthBlock        = errors.New("ancient birth block")
	errExhaustsResources        = errors.New("exhausts resources")
	errMandatoryDispatchError   = errors.New("mandatory dispatch error")
	errInvalidMandatoryDispatch = errors.New("invalid mandatory dispatch")
)

// Call The call of the transaction is not expected
type Call struct{}

// Index Returns VDT index
func (Call) Index() uint { return 0 }

// Payment General error to do with the inability to pay some fees (e.g. account balance too low)
type Payment struct{}

// Index Returns VDT index
func (Payment) Index() uint { return 1 }

// Future General error to do with the transaction not yet being valid (e.g. nonce too high)
type Future struct{}

// Index Returns VDT index
func (Future) Index() uint { return 2 }

// Stale General error to do with the transaction being outdated (e.g. nonce too low)
type Stale struct{}

// Index Returns VDT index
func (Stale) Index() uint { return 3 }

// BadProof General error to do with the transaction’s proofs (e.g. signature)
type BadProof struct{}

// Index Returns VDT index
func (BadProof) Index() uint { return 4 }

// AncientBirthBlock The transaction birth block is ancient
type AncientBirthBlock struct{}

// Index Returns VDT index
func (AncientBirthBlock) Index() uint { return 5 }

// ExhaustsResources The transaction would exhaust the resources of current block
type ExhaustsResources struct{}

// Index Returns VDT index
func (ExhaustsResources) Index() uint { return 6 }

var invalidCustom InvalidCustom

// InvalidCustom Any other custom invalid validity that is not covered
type InvalidCustom uint8

// Index Returns VDT index
func (InvalidCustom) Index() uint { return 7 }

// BadMandatory An extrinsic with a Mandatory dispatch resulted in Error
type BadMandatory struct{}

// Index Returns VDT index
func (BadMandatory) Index() uint { return 8 }

// MandatoryDispatch A transaction with a mandatory dispatch
type MandatoryDispatch struct{}

// Index Returns VDT index
func (MandatoryDispatch) Index() uint { return 9 }

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (i *InvalidTransaction) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*i)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*i = InvalidTransaction(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (i *InvalidTransaction) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*i)
	return vdt.Value()
}

// NewInvalidTransaction is constructor for InvalidTransaction
func NewInvalidTransaction() InvalidTransaction {
	vdt, err := scale.NewVaryingDataType(Call{}, Payment{}, Future{}, Stale{}, BadProof{}, AncientBirthBlock{},
		ExhaustsResources{}, invalidCustom, BadMandatory{}, MandatoryDispatch{})
	if err != nil {
		panic(err)
	}
	return InvalidTransaction(vdt)
}

func (i *InvalidTransaction) Error() string {
	switch val := i.Value().(type) {
	case Call:
		return errUnexpectedTxCall.Error()
	case Payment:
		return errInvalidPayment.Error()
	case Future:
		return errInvalidTransaction.Error()
	case Stale:
		return errOutdatedTransaction.Error()
	case BadProof:
		return errBadProof.Error()
	case AncientBirthBlock:
		return errAncientBirthBlock.Error()
	case ExhaustsResources:
		return errExhaustsResources.Error()
	case InvalidCustom:
		return newUnknownError(val).Error()
	case BadMandatory:
		return errMandatoryDispatchError.Error()
	case MandatoryDispatch:
		return errInvalidMandatoryDispatch.Error()
	}

	return errInvalidResult.Error()
}
