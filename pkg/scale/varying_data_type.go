// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

// EncodeVaryingDataType is used in VaryingDataType. It contains the methods required
// for encoding.
type EncodeVaryingDataType interface {
	IndexValue() (index uint, value any, err error)
	Value() (value any, err error)
	ValueAt(index uint) (value any, err error)
}

// VaryingDataType is analogous to a rust enum.  Name is taken from polkadot spec.
type VaryingDataType interface {
	EncodeVaryingDataType
	SetValue(value any) (err error)
}
