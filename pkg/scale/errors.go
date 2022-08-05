// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import "errors"

var (
	ErrUnsupportedDestination            = errors.New("must be a pointer to a destination, unsupported destination")
	errDecodeBoolFailed                  = errors.New("decodeBool failed")
	errUnsupportedType                   = errors.New("unsupported type")
	errUnsupportedResult                 = errors.New("unsupported result")
	errUnsupportedOption                 = errors.New("unsupported option")
	errFindVDT                           = errors.New("unable to find VaryingDataTypeValue with index")
	errDecodeInteger                     = errors.New("could not decode invalid integer")
	errEncodeFixedWidthIntFailed         = errors.New("failed to encode fixed width int")
	errEncodeBigIntFailed                = errors.New("failed to encode big int")
	errEncodeUint128Failed               = errors.New("failed to encode uint128")
	errEncodeResultFailed                = errors.New("failed to encode result")
	ErrResultAlreadySet                  = errors.New("result already has an assigned value")
	ErrAddVaryingDataTypeValueNotInCache = errors.New("failed to add VaryingDataTypeValue not in cache")
	ErrMustProvideVaryingDataTypeValue   = errors.New("must provide at least one VaryingDataTypeValue")
)
