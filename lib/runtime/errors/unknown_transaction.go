// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package errors

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// UnknownTransaction is child VDT of TransactionValidityError
type UnknownTransaction scale.VaryingDataType

// Index fulfils the VaryingDataTypeValue interface.  T
func (UnknownTransaction) Index() uint {
	return 1
}

// ValidityCannotLookup Could not look up some information that is required to validate the transaction
type ValidityCannotLookup struct{}

// Index Returns VDT index
func (ValidityCannotLookup) Index() uint { return 0 }

// NoUnsignedValidator No validator found for the given unsigned transaction
type NoUnsignedValidator struct{}

// Index Returns VDT index
func (NoUnsignedValidator) Index() uint { return 1 }

// UnknownCustom Any other custom unknown validity that is not covered
type UnknownCustom uint8

// Index Returns VDT index
func (UnknownCustom) Index() uint { return 2 }

func newUnknownError(data scale.VaryingDataTypeValue) error {
	return fmt.Errorf("unknown error: %v", data)
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

// Value will return value from underying VaryingDataType
func (u *UnknownTransaction) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*u)
	return vdt.Value()
}

// NewUnknownTransaction is constructor for Unknown
func NewUnknownTransaction() UnknownTransaction {
	vdt, err := scale.NewVaryingDataType(ValidityCannotLookup{}, NoUnsignedValidator{}, UnknownCustom(0))
	if err != nil {
		panic(err)
	}
	return UnknownTransaction(vdt)
}

func (u *UnknownTransaction) Error() string {
	switch val := u.Value().(type) {
	case ValidityCannotLookup:
		return "lookup failed"
	case NoUnsignedValidator:
		return "validator not found"
	case UnknownCustom:
		return newUnknownError(val).Error()
	default:
		panic("unknownTransaction: invalid error value")
	}
}
