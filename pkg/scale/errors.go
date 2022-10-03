// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import "errors"

var (
	ErrUnsupportedDestination            = errors.New("must be a pointer to a destination, unsupported destination")
	errDecodeBool                        = errors.New("decodeBool failed")
	ErrUnsupportedType                   = errors.New("unsupported type")
	errUnsupportedResult                 = errors.New("unsupported result")
	errUnsupportedOption                 = errors.New("unsupported option")
	errFindVDT                           = errors.New("unable to find VaryingDataTypeValue with index")
	errUint128IsNil                      = errors.New("uint128 in nil")
	errResultNotSet                      = errors.New("result not set")
	ErrResultAlreadySet                  = errors.New("result already has an assigned value")
	ErrAddVaryingDataTypeValueNotInCache = errors.New("failed to add VaryingDataTypeValue not in cache")
	ErrMustProvideVaryingDataTypeValue   = errors.New("must provide at least one VaryingDataTypeValue")
	errBigIntIsNil                       = errors.New("big int is nil")
	ErrVaryingDataTypeNotSet             = errors.New("varying data type not set")
)
