// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// UnknownTransaction is the child VDT of TransactionValidityError
type UnknownTransaction struct {
	inner any
}
type UnknownTransactionValues interface {
	ValidityCannotLookup | NoUnsignedValidator | UnknownCustom
}

func setUnknownTransaction[Value UnknownTransactionValues](mvdt *UnknownTransaction, value Value) {
	mvdt.inner = value
}

func (mvdt *UnknownTransaction) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ValidityCannotLookup:
		setUnknownTransaction(mvdt, value)
		return
	case NoUnsignedValidator:
		setUnknownTransaction(mvdt, value)
		return
	case UnknownCustom:
		setUnknownTransaction(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt UnknownTransaction) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case ValidityCannotLookup:
		return 0, mvdt.inner, nil
	case NoUnsignedValidator:
		return 1, mvdt.inner, nil
	case UnknownCustom:
		return 2, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt UnknownTransaction) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt UnknownTransaction) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return ValidityCannotLookup{}, nil
	case 1:
		return NoUnsignedValidator{}, nil
	case 2:
		return UnknownCustom(0), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

func (u UnknownTransaction) Error() string { //skipcq: GO-W1029
	value, err := u.Value()
	if err != nil {
		return fmt.Sprintf("getting unknown transaction value: %s", err)
	}
	err, ok := value.(error)
	if !ok {
		panic(fmt.Sprintf("%T does not implement the error type", value))
	}
	return err.Error()
}

// NewUnknownTransaction is constructor for UnknownTransaction
func NewUnknownTransaction() UnknownTransaction {
	return UnknownTransaction{}
}

// ValidityCannotLookup Could not look up some information that is required to validate the transaction
type ValidityCannotLookup struct{}

func (v ValidityCannotLookup) String() string { return v.Error() }

// Error returns the error message associated with the ValidityCannotLookup
func (ValidityCannotLookup) Error() string {
	return "lookup failed"
}

// NoUnsignedValidator No validator found for the given unsigned transaction
type NoUnsignedValidator struct{}

func (n NoUnsignedValidator) String() string { return n.Error() }

// Error returns the error message associated with the NoUnsignedValidator
func (NoUnsignedValidator) Error() string {
	return "validator not found"
}

// UnknownCustom Any other custom unknown validity that is not covered
type UnknownCustom uint8

// Error returns the error message associated with the UnknownCustom
func (m UnknownCustom) Error() string {
	return newUnknownError(m).Error()
}

func newUnknownError(data any) error {
	return fmt.Errorf("unknown error: %v", data)
}
