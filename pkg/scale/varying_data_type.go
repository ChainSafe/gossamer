// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

type EncodeVaryingDataType interface {
	IndexValue() (index uint, value any, err error)
	Value() (value any, err error)
	ValueAt(index uint) (value any, err error)
}

type VaryingDataType interface {
	EncodeVaryingDataType
	SetValue(value any) (err error)
}
