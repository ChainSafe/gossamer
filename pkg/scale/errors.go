// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import "errors"

var (
	ErrUnsupportedDestination            = errors.New("must be a pointer to a destination, unsupported destination")
	ErrDecodeBool                        = errors.New("decodeBool failed")
	ErrUnsupportedType                   = errors.New("unsupported type")
	ErrUnsupportedResult                 = errors.New("unsupported result")
	ErrUnsupportedOption                 = errors.New("unsupported option")
	ErrFindVDT                           = errors.New("unable to find VaryingDataTypeValue with index")
	ErrDecodeInteger                     = errors.New("could not decode invalid integer")
	ErrEncodeFixedWidthInt               = errors.New("failed to encode fixed width int")
	ErrEncodeBigInt                      = errors.New("failed to encode big int")
	ErrEncodeUint128                     = errors.New("failed to encode uint128")
	ErrEncodeResult                      = errors.New("failed to encode result")
	ErrResultAlreadySet                  = errors.New("result already has an assigned value")
	ErrAddVaryingDataTypeValueNotInCache = errors.New("failed to add VaryingDataTypeValue not in cache")
	ErrMustProvideVaryingDataTypeValue   = errors.New("must provide at least one VaryingDataTypeValue")
	ErrBigIntIsNil                       = errors.New("big int is nil")
)
