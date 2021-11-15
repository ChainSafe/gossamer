// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"fmt"
)

// VaryingDataTypeValue is used to represent scale encodable types of an associated VaryingDataType
type VaryingDataTypeValue interface {
	Index() uint
}

// VaryingDataTypeSlice is used to represent []VaryingDataType.  SCALE requires knowledge
// of the underlying data, so it is required to have the VaryingDataType required for decoding
type VaryingDataTypeSlice struct {
	VaryingDataType
	Types []VaryingDataType
}

// Add takes variadic parameter values to add VaryingDataTypeValue(s)
func (vdts *VaryingDataTypeSlice) Add(values ...VaryingDataTypeValue) (err error) {
	for _, val := range values {
		copied := vdts.VaryingDataType
		err = copied.Set(val)
		if err != nil {
			return
		}
		vdts.Types = append(vdts.Types, copied)
	}
	return
}

// NewVaryingDataTypeSlice is constructor for VaryingDataTypeSlice
func NewVaryingDataTypeSlice(vdt VaryingDataType) (vdts VaryingDataTypeSlice) {
	vdts.VaryingDataType = vdt
	vdts.Types = make([]VaryingDataType, 0)
	return
}

func mustNewVaryingDataTypeSliceAndSet(vdt VaryingDataType, values ...VaryingDataTypeValue) (vdts VaryingDataTypeSlice) {
	vdts = NewVaryingDataTypeSlice(vdt)
	if err := vdts.Add(values...); err != nil {
		panic(err)
	}
	return
}

// VaryingDataType is analogous to a rust enum.  Name is taken from polkadot spec.
type VaryingDataType struct {
	value VaryingDataTypeValue
	cache map[uint]VaryingDataTypeValue
}

// Set will set the VaryingDataType value
func (vdt *VaryingDataType) Set(value VaryingDataTypeValue) (err error) {
	_, ok := vdt.cache[value.Index()]
	if !ok {
		err = fmt.Errorf("unable to append VaryingDataTypeValue: %T, not in cache", value)
		return
	}
	vdt.value = value
	return
}

// Value returns value stored in vdt
func (vdt *VaryingDataType) Value() VaryingDataTypeValue {
	return vdt.value
}

// NewVaryingDataType is constructor for VaryingDataType
func NewVaryingDataType(values ...VaryingDataTypeValue) (vdt VaryingDataType, err error) {
	if len(values) == 0 {
		err = fmt.Errorf("must provide atleast one VaryingDataTypeValue")
		return
	}
	vdt.cache = make(map[uint]VaryingDataTypeValue)
	for _, value := range values {
		_, ok := vdt.cache[value.Index()]
		if ok {
			err = fmt.Errorf("duplicate index with VaryingDataType: %T with index: %d", value, value.Index())
			return
		}
		vdt.cache[value.Index()] = value
	}
	return
}

// MustNewVaryingDataType is constructor for VaryingDataType
func MustNewVaryingDataType(values ...VaryingDataTypeValue) (vdt VaryingDataType) {
	vdt, err := NewVaryingDataType(values...)
	if err != nil {
		panic(err)
	}
	return
}
