// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import "errors"

var (
	ErrUnsupportedDST                  = errors.New("must be a pointer to a destination, unsupported dst")
	errDecodeBoolFailed                = errors.New("decodeBool failed")
	errUnsupportedType                 = errors.New("unsupported type")
	errUnsupportedResult               = errors.New("unsupported result")
	errUnsupportedOption               = errors.New("unsupported option")
	errUnableToFindVDT                 = errors.New("unable to find VaryingDataTypeValue with index")
	errDecodeInteger                   = errors.New("could not decode invalid integer")
	errEncodeFixedWidthIntFailed       = errors.New("failed to encode fixed width int")
	errEncodeBigIntFailed              = errors.New("failed to encode big int")
	errEncodeUint128Failed             = errors.New("failed to encode uint128")
	errEncodeResultFailed              = errors.New("failed to encode result")
	ErrResultAlreadySet                = errors.New("result already has an assigned value")
	ErrAddVaryingDataTypeValue         = errors.New("failed to add VaryingDataTypeValue")
	ErrMustProvideVaryingDataTypeValue = errors.New("must provide at least one VaryingDataTypeValue")
)
