// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import "errors"

var (
	ErrUnsupportedDestination          = errors.New("must be a non-nil pointer to a destination")
	errDecodeBool                      = errors.New("invalid byte for bool")
	ErrUnsupportedType                 = errors.New("unsupported type")
	ErrUnsupportedResult               = errors.New("unsupported result")
	errUnsupportedOption               = errors.New("unsupported option")
	ErrUnknownVaryingDataTypeValue     = errors.New("unable to find VaryingDataTypeValue with index")
	errUint128IsNil                    = errors.New("uint128 in nil")
	ErrResultNotSet                    = errors.New("result not set")
	ErrResultAlreadySet                = errors.New("result already has an assigned value")
	ErrUnsupportedVaryingDataTypeValue = errors.New("unsupported varying data type value")
	ErrMustProvideVaryingDataTypeValue = errors.New("must provide at least one VaryingDataTypeValue")
	errBigIntIsNil                     = errors.New("big int is nil")
	ErrVaryingDataTypeNotSet           = errors.New("varying data type not set")
	ErrUnsupportedCustomPrimitive      = errors.New("unsupported type for custom primitive")
	ErrInvalidScaleIndex               = errors.New("invalid scale index")
)
