// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// InvalidTransaction is a child VDT of TransactionValidityError
type InvalidTransaction scale.VaryingDataType

// Index returns the VDT index
func (InvalidTransaction) Index() uint { //skipcq: GO-W1029
	return 0
}

func (i InvalidTransaction) String() string { return i.Error() } //skipcq: GO-W1029

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (i *InvalidTransaction) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*i)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*i = InvalidTransaction(vdt)
	return nil
}

// Value will return the value from the underying VaryingDataType
func (i *InvalidTransaction) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(*i)
	return vdt.Value()
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
	vdt := scale.MustNewVaryingDataType(Call{}, Payment{}, Future{}, Stale{}, BadProof{}, AncientBirthBlock{},
		ExhaustsResources{}, InvalidCustom(0), BadMandatory{}, MandatoryDispatch{}, BadSigner{})
	return InvalidTransaction(vdt)
}

// Call The call of the transaction is not expected
type Call struct{}

// Index returns the VDT index
func (Call) Index() uint { return 0 }

func (c Call) String() string { return c.Error() }

// Error returns the error message associated with the Call
func (Call) Error() string {
	return "call of the transaction is not expected"
}

// Payment General error to do with the inability to pay some fees (e.g. account balance too low)
type Payment struct{}

// Index returns the VDT index
func (Payment) Index() uint { return 1 }

func (p Payment) String() string { return p.Error() }

// Error returns the error message associated with the Payment
func (Payment) Error() string {
	return "invalid payment"
}

// Future General error to do with the transaction not yet being valid (e.g. nonce too high)
type Future struct{}

// Index returns the VDT index
func (Future) Index() uint { return 2 }

func (f Future) String() string { return f.Error() }

// Error returns the error message associated with the Future
func (Future) Error() string {
	return "invalid transaction"
}

// Stale General error to do with the transaction being outdated (e.g. nonce too low)
type Stale struct{}

// Index returns the VDT index
func (Stale) Index() uint { return 3 }

func (s Stale) String() string { return s.Error() }

// Error returns the error message associated with the Stale
func (Stale) Error() string {
	return "outdated transaction"
}

// BadProof General error to do with the transaction’s proofs (e.g. signature)
type BadProof struct{}

// Index returns the VDT index
func (BadProof) Index() uint { return 4 }

func (b BadProof) String() string { return b.Error() }

// Error returns the error message associated with the BadProof
func (BadProof) Error() string {
	return "bad proof"
}

// AncientBirthBlock The transaction birth block is ancient
type AncientBirthBlock struct{}

// Index returns the VDT index
func (AncientBirthBlock) Index() uint { return 5 }

func (a AncientBirthBlock) String() string { return a.Error() }

// Error returns the error message associated with the AncientBirthBlock
func (AncientBirthBlock) Error() string {
	return "ancient birth block"
}

// ExhaustsResources The transaction would exhaust the resources of current block
type ExhaustsResources struct{}

// Index returns the VDT index
func (ExhaustsResources) Index() uint { return 6 }

func (e ExhaustsResources) String() string { return e.Error() }

// Error returns the error message associated with the ExhaustsResources
func (ExhaustsResources) Error() string {
	return "exhausts resources"
}

// InvalidCustom Any other custom invalid validity that is not covered
type InvalidCustom uint8

// Index returns the VDT index
func (InvalidCustom) Index() uint { return 7 }

func (i InvalidCustom) String() string { return i.Error() }

// Error returns the error message associated with the InvalidCustom
func (i InvalidCustom) Error() string {
	return newUnknownError(i).Error()
}

// BadMandatory An extrinsic with a Mandatory dispatch resulted in Error
type BadMandatory struct{}

// Index returns the VDT index
func (BadMandatory) Index() uint { return 8 }

func (b BadMandatory) String() string { return b.Error() }

// Error returns the error message associated with the BadMandatory
func (BadMandatory) Error() string {
	return "mandatory dispatch error"
}

// MandatoryDispatch A transaction with a mandatory dispatch
type MandatoryDispatch struct{}

// Index returns the VDT index
func (MandatoryDispatch) Index() uint { return 9 }

func (m MandatoryDispatch) String() string { return m.Error() }

// Error returns the error message associated with the MandatoryDispatch
func (MandatoryDispatch) Error() string {
	return "invalid mandatory dispatch"
}

// BadSigner A transaction with a mandatory dispatch
type BadSigner struct{}

// Index returns VDT index
func (BadSigner) Index() uint { return 10 }

func (b BadSigner) String() string { return b.Error() }

// Error returns the error message associated with the MandatoryDispatch
func (BadSigner) Error() string {
	return "invalid signing address"
}
