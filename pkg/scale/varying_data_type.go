// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

// // VaryingDataTypeSlice is used to represent []VaryingDataType. SCALE requires knowledge
// // of the underlying data, so it is required to have the VaryingDataType required for decoding
// type VaryingDataTypeSlice struct {
// 	VaryingDataType
// 	Types []VaryingDataType
// }

// // Add takes variadic parameter values to add VaryingDataTypeValue(s)
// func (vdts *VaryingDataTypeSlice) Add(values ...VaryingDataTypeValue) (err error) { //skipcq: GO-W1029
// 	for _, val := range values {
// 		copied := vdts.VaryingDataType
// 		err = copied.SetValue(val)
// 		if err != nil {
// 			err = fmt.Errorf("setting VaryingDataTypeValue: %w", err)
// 			return
// 		}
// 		vdts.Types = append(vdts.Types, copied)
// 	}
// 	return
// }

// func (vdts VaryingDataTypeSlice) String() string { //skipcq: GO-W1029
// 	stringTypes := make([]string, len(vdts.Types))
// 	for i, vdt := range vdts.Types {
// 		stringTypes[i] = vdt.String()
// 	}
// 	return "[" + strings.Join(stringTypes, ", ") + "]"
// }

// // NewVaryingDataTypeSlice is constructor for VaryingDataTypeSlice
// func NewVaryingDataTypeSlice(vdt VaryingDataType) (vdts VaryingDataTypeSlice) {
// 	vdts.VaryingDataType = vdt
// 	vdts.Types = make([]VaryingDataType, 0)
// 	return
// }

// func mustNewVaryingDataTypeSliceAndSet(vdt VaryingDataType,
//
//		values ...VaryingDataTypeValue) (vdts VaryingDataTypeSlice) {
//		vdts = NewVaryingDataTypeSlice(vdt)
//		if err := vdts.Add(values...); err != nil {
//			panic(fmt.Sprintf("adding varying data type value: %s", err))
//		}
//		return
//	}
//

// VaryingDataType is analogous to a rust enum.  Name is taken from polkadot spec.
type VaryingDataType interface {
	SetValue(value any) (err error)
	IndexValue() (index uint, value any, err error)
	Value() (value any, err error)
	ValueAt(index uint) (value any, err error)
}
