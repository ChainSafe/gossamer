// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale_test

import (
	"fmt"
	"reflect"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ParentVDT is a VaryingDataType that consists of multiple nested VaryingDataType
// instances (aka. a rust enum containing multiple enum options)
type ParentVDT scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (pvdt *ParentVDT) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*pvdt)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	// store original ParentVDT with VaryingDataType that has been set
	*pvdt = ParentVDT(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (pvdt *ParentVDT) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*pvdt)
	return vdt.Value()
}

// NewParentVDT is constructor for ParentVDT
func NewParentVDT() ParentVDT {
	// use standard VaryingDataType constructor to construct a VaryingDataType
	vdt, err := scale.NewVaryingDataType(NewChildVDT(), NewOtherChildVDT())
	if err != nil {
		panic(err)
	}
	// cast to ParentVDT
	return ParentVDT(vdt)
}

// ChildVDT type is used as a VaryingDataTypeValue for ParentVDT
type ChildVDT scale.VaryingDataType

// Index fulfils the VaryingDataTypeValue interface.  T
func (cvdt ChildVDT) Index() uint {
	return 1
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (cvdt *ChildVDT) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*cvdt)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	// store original ParentVDT with VaryingDataType that has been set
	*cvdt = ChildVDT(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (cvdt *ChildVDT) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*cvdt)
	return vdt.Value()
}

// NewChildVDT is constructor for ChildVDT
func NewChildVDT() ChildVDT {
	// use standard VaryingDataType constructor to construct a VaryingDataType
	// constarined to types ChildInt16, ChildStruct, and ChildString
	vdt, err := scale.NewVaryingDataType(ChildInt16(0), ChildStruct{}, ChildString(""))
	if err != nil {
		panic(err)
	}
	// cast to ParentVDT
	return ChildVDT(vdt)
}

// OtherChildVDT type is used as a VaryingDataTypeValue for ParentVDT
type OtherChildVDT scale.VaryingDataType

// Index fulfils the VaryingDataTypeValue interface.
func (ocvdt OtherChildVDT) Index() uint {
	return 2
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (cvdt *OtherChildVDT) Set(val scale.VaryingDataTypeValue) (err error) { //nolint:revive
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*cvdt)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	// store original ParentVDT with VaryingDataType that has been set
	*cvdt = OtherChildVDT(vdt)
	return
}

// NewOtherChildVDT is constructor for OtherChildVDT
func NewOtherChildVDT() OtherChildVDT {
	// use standard VaryingDataType constructor to construct a VaryingDataType
	// constarined to types ChildInt16 and ChildStruct
	vdt, err := scale.NewVaryingDataType(ChildInt16(0), ChildStruct{}, ChildString(""))
	if err != nil {
		panic(err)
	}
	// cast to ParentVDT
	return OtherChildVDT(vdt)
}

// ChildInt16 is used as a VaryingDataTypeValue for ChildVDT and OtherChildVDT
type ChildInt16 int16

// Index fulfils the VaryingDataTypeValue interface.  The ChildVDT type is used as a
// VaryingDataTypeValue for ParentVDT
func (ci ChildInt16) Index() uint {
	return 1
}

// ChildStruct is used as a VaryingDataTypeValue for ChildVDT and OtherChildVDT
type ChildStruct struct {
	A string
	B bool
}

// Index fulfils the VaryingDataTypeValue interface
func (cs ChildStruct) Index() uint {
	return 2
}

// ChildString is used as a VaryingDataTypeValue for ChildVDT and OtherChildVDT
type ChildString string

// Index fulfils the VaryingDataTypeValue interface
func (cs ChildString) Index() uint {
	return 3
}

func Example() {
	parent := NewParentVDT()

	// populate parent with ChildVDT
	child := NewChildVDT()
	child.Set(ChildInt16(888))
	err := parent.Set(child)
	if err != nil {
		panic(err)
	}

	// validate ParentVDT.Value()
	fmt.Printf("parent.Value(): %+v\n", parent.Value())
	// should cast to ChildVDT, since that was set earlier
	valChildVDT := parent.Value().(ChildVDT)
	// validate ChildVDT.Value() as ChildInt16(888)
	fmt.Printf("child.Value(): %+v\n", valChildVDT.Value())

	// marshal into scale encoded bytes
	bytes, err := scale.Marshal(parent)
	if err != nil {
		panic(err)
	}
	fmt.Printf("bytes: % x\n", bytes)

	// unmarshal into another ParentVDT
	dstParent := NewParentVDT()
	err = scale.Unmarshal(bytes, &dstParent)
	if err != nil {
		panic(err)
	}
	// assert both ParentVDT instances are the same
	fmt.Println(reflect.DeepEqual(parent, dstParent))

	// Output:
	// parent.Value(): {value:888 cache:map[1:0 2:{A: B:false} 3:]}
	// child.Value(): 888
	// bytes: 01 01 78 03
	// true
}
