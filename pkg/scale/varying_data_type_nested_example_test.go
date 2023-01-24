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
func (pvdt *ParentVDT) Value() (val scale.VaryingDataTypeValue, err error) {
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
func (ChildVDT) Index() uint { //skipcq: GO-W1029
	return 1
}

func (cvdt ChildVDT) String() string { //skipcq: GO-W1029
	value, err := cvdt.Value()
	if err != nil {
		return "ChildVDT()"
	}
	return fmt.Sprintf("ChildVDT(%s)", value)
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (cvdt *ChildVDT) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
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
func (cvdt *ChildVDT) Value() (val scale.VaryingDataTypeValue, err error) { //skipcq: GO-W1029
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
func (OtherChildVDT) Index() uint { //skipcq: GO-W1029
	return 2
}

func (cvdt OtherChildVDT) String() string { //skipcq: GO-W1029
	vdt := scale.VaryingDataType(cvdt)
	vdtPtr := &vdt
	value, err := vdtPtr.Value()
	if err != nil {
		return "OtherChildVDT()"
	}
	return fmt.Sprintf("OtherChildVDT(%s)", value)
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (cvdt *OtherChildVDT) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
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
func (ChildInt16) Index() uint {
	return 1
}

func (c ChildInt16) String() string { return fmt.Sprintf("ChildInt16(%d)", c) }

// ChildStruct is used as a VaryingDataTypeValue for ChildVDT and OtherChildVDT
type ChildStruct struct {
	A string
	B bool
}

// Index fulfils the VaryingDataTypeValue interface
func (ChildStruct) Index() uint {
	return 2
}

func (c ChildStruct) String() string {
	return fmt.Sprintf("ChildStruct{A=%s, B=%t}", c.A, c.B)
}

// ChildString is used as a VaryingDataTypeValue for ChildVDT and OtherChildVDT
type ChildString string

// Index fulfils the VaryingDataTypeValue interface
func (ChildString) Index() uint {
	return 3
}

func (c ChildString) String() string { return fmt.Sprintf("ChildString(%s)", string(c)) }

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
	parentVal, err := parent.Value()
	if err != nil {
		panic(err)
	}
	fmt.Printf("parent.Value(): %+v\n", parentVal)
	// should cast to ChildVDT, since that was set earlier
	valChildVDT := parentVal.(ChildVDT)
	// validate ChildVDT.Value() as ChildInt16(888)
	childVdtValue, err := valChildVDT.Value()
	if err != nil {
		panic(err)
	}
	fmt.Printf("child.Value(): %+v\n", childVdtValue)

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
	// parent.Value(): ChildVDT(ChildInt16(888))
	// child.Value(): ChildInt16(888)
	// bytes: 01 01 78 03
	// true
}
