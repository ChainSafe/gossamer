// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package transaction_validity

import (
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// UnknownTransaction is child VDT of TransactionValidityError
type UnknownTransaction scale.VaryingDataType

// Index fulfils the VaryingDataTypeValue interface.  T
func (u UnknownTransaction) Index() uint {
	return 1
}

var (
	errLookupFailed      = errors.New("lookup failed")
	errValidatorNotFound = errors.New("validator not found")
	unknownCustom        UnknownCustom
)

// ValidityCannotLookup Could not lookup some information that is required to validate the transaction
type ValidityCannotLookup struct{}

// Index Returns VDT index
func (err ValidityCannotLookup) Index() uint { return 0 }

// NoUnsignedValidator No validator found for the given unsigned transaction
type NoUnsignedValidator struct{}

// Index Returns VDT index
func (err NoUnsignedValidator) Index() uint { return 1 }

// UnknownCustom Any other custom unknown validity that is not covered
type UnknownCustom uint8

// Index Returns VDT index
func (err UnknownCustom) Index() uint { return 2 }

func newUnknownError(data scale.VaryingDataTypeValue) error {
	return fmt.Errorf("unknown error: %d", data)
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (u *UnknownTransaction) Set(val scale.VaryingDataTypeValue) (err error) { //nolint:revive
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*u)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	// store original ParentVDT with VaryingDataType that has been set
	*u = UnknownTransaction(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (u *UnknownTransaction) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*u)
	return vdt.Value()
}

// NewUnknownTransaction is constructor for Unknown
func NewUnknownTransaction() UnknownTransaction {
	// use standard VaryingDataType constructor to construct a VaryingDataType
	// constarined to types ChildInt16 and ChildStruct
	vdt, err := scale.NewVaryingDataType(ValidityCannotLookup{}, NoUnsignedValidator{}, unknownCustom)
	if err != nil {
		panic(err)
	}
	// cast to ParentVDT
	return UnknownTransaction(vdt)
}

func (u *UnknownTransaction) Error() error {
	switch val := u.Value().(type) {
	// UnknownTransaction Error
	case ValidityCannotLookup:
		return errLookupFailed
	case NoUnsignedValidator:
		return errValidatorNotFound
	case UnknownCustom:
		return newUnknownError(val)
	}

	return errInvalidResult
}
