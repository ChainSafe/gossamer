// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import "fmt"

var (
	ErrUnsupportedDestination          = fmt.Errorf("must be a non-nil pointer to a destination")
	errDecodeBool                      = fmt.Errorf("invalid byte for bool")
	ErrUnsupportedType                 = fmt.Errorf("unsupported type")
	ErrUnsupportedResult               = fmt.Errorf("unsupported result")
	errUnsupportedOption               = fmt.Errorf("unsupported option")
	errUnknownVaryingDataTypeValue     = fmt.Errorf("unable to find VaryingDataTypeValue with index")
	errUint128IsNil                    = fmt.Errorf("uint128 in nil")
	ErrResultNotSet                    = fmt.Errorf("result not set")
	ErrResultAlreadySet                = fmt.Errorf("result already has an assigned value")
	ErrUnsupportedVaryingDataTypeValue = fmt.Errorf("unsupported VaryingDataTypeValue")
	ErrMustProvideVaryingDataTypeValue = fmt.Errorf("must provide at least one VaryingDataTypeValue")
	errBigIntIsNil                     = fmt.Errorf("big int is nil")
	ErrVaryingDataTypeNotSet           = fmt.Errorf("varying data type not set")
	ErrUnsupportedCustomPrimitive      = fmt.Errorf("unsupported type for custom primitive")
)
