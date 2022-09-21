// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// UnknownTransaction is the child VDT of TransactionValidityError
type UnknownTransaction scale.VaryingDataType

// Index returns the VDT index
func (UnknownTransaction) Index() uint {
	return 1
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (u *UnknownTransaction) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*u)
	err = vdt.Set(val)
	if err != nil {
		return err
	}
	*u = UnknownTransaction(vdt)
	return nil
}

// Value will return value from the underying VaryingDataType
func (u *UnknownTransaction) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*u)
	return vdt.Value()
}

func (u UnknownTransaction) Error() string {
	value := u.Value()
	if value == nil {
		return "unknownTransaction hasn't been set"
	}
	err, ok := value.(error)
	if !ok {
		panic(fmt.Sprintf("%T does not implement the error type", value))
	}
	return err.Error()
}

// NewUnknownTransaction is constructor for Unknown
func NewUnknownTransaction() UnknownTransaction {
	vdt, err := scale.NewVaryingDataType(ValidityCannotLookup{}, NoUnsignedValidator{}, UnknownCustom(0))
	if err != nil {
		panic(err)
	}
	return UnknownTransaction(vdt)
}

// ValidityCannotLookup Could not look up some information that is required to validate the transaction
type ValidityCannotLookup struct{}

// Index returns the VDT index
func (ValidityCannotLookup) Index() uint { return 0 }

// Error returns the error message associated with the ValidityCannotLookup
func (ValidityCannotLookup) Error() string {
	return "lookup failed"
}

// NoUnsignedValidator No validator found for the given unsigned transaction
type NoUnsignedValidator struct{}

// Index returns the VDT index
func (NoUnsignedValidator) Index() uint { return 1 }

// Error returns the error message associated with the NoUnsignedValidator
func (NoUnsignedValidator) Error() string {
	return "validator not found"
}

// UnknownCustom Any other custom unknown validity that is not covered
type UnknownCustom uint8

// Index returns the VDT index
func (UnknownCustom) Index() uint { return 2 }

// Error returns the error message associated with the UnknownCustom
func (m UnknownCustom) Error() string {
	return newUnknownError(m).Error()
}

func newUnknownError(data scale.VaryingDataTypeValue) error {
	return fmt.Errorf("unknown error: %v", data)
}
