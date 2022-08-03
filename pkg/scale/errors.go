// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import "errors"

var (
	ErrVaryingDataTypeNotSet = errors.New("VaryingDataTypeValue has not been set")
	ErrUnsupportedDST                    = errors.New("must be a pointer to a destination, unsupported dst")
	ErrUnmarshalFailed                   = errors.New("unmarshal failed")
	ErrIndirectFailed                    = errors.New("indirect failed")
	errDecodeBigIntFailed                = errors.New("decodeBigInt failed")
	errDecodeUint128Failed               = errors.New("decodeUint128 failed")
	errDecodeUintFailed                  = errors.New("decodeUint failed")
	errDecodeFixedWidthIntFailed         = errors.New("decodeFixedWidthInt failed")
	errDecodeBytesFailed                 = errors.New("decodeBytes failed")
	errDecodeBoolFailed                  = errors.New("decodeBool failed")
	errDecodeResultFailed                = errors.New("decodeResult failed")
	errDecodeVaryingDataTypeFailed       = errors.New("decodeVaryingDataType failed")
	errDecodeVaryingDataTypeSliceFailed  = errors.New("decodeVaryingDataTypeSlice failed")
	errDecodeCustomPrimitiveFailed       = errors.New("decodeCustomPrimitive failed")
	errDecodePointerFailed               = errors.New("decodePointer failed")
	errDecodeCustomVaryingDataTypeFailed = errors.New("decodeCustomVaryingDataType failed")
	errDecodeStructFailed                = errors.New("decodeStruct failed")
	errDecodeArrayFailed                 = errors.New("decodeArray failed")
	errDecodeSliceFailed                 = errors.New("decodeSlice failed")
	errUnsupportedType                   = errors.New("unsupported type")
	ErrReadByteFailed                    = errors.New("ReadByte failed")
	ErrSetFailed                         = errors.New("Set failed")
	errUnsupportedResult                 = errors.New("unsupported result")
	errUnsupportedOption                 = errors.New("unsupported option")
	errDecodeLengthFailed                = errors.New("decodeLength failed")
	errUnableToFindVDT                   = errors.New("unable to find VaryingDataTypeValue with index")
	errCacheFieldScaleIndicesFailed      = errors.New("cache.fieldScaleIndices failed")
	errDecodeSmallIntFailed              = errors.New("decodeSmallInt failed")
	errDecodeIntegerFailed               = errors.New("could not decode invalid integer")

	ErrMarshalFailed                     = errors.New("marshal failed")
	errEncodeIntFailed                   = errors.New("failed to encode int")
	errEncodeUintFailed                  = errors.New("failed to encode uint")
	errEncodeFixedWidthIntFailed         = errors.New("failed to encode fixed width int")
	errEncodeBigIntFailed                = errors.New("failed to encode big int")
	errEncodeUint128Failed               = errors.New("failed to encode uint128")
	errEncodeBytesFailed                 = errors.New("failed to encode bytes")
	errEncodeStringFailed                = errors.New("failed to encode string")
	errEncodeBoolFailed                  = errors.New("failed to encode bool")
	errEncodeResultFailed                = errors.New("failed to encode result")
	errEncodeVaryingDataTypeFailed       = errors.New("failed to encode varying data type")
	errEncodeVaryingDataTypeSliceFailed  = errors.New("failed to encode varying data type slice")
	errEncodeCustomVaryingDataTypeFailed = errors.New("failed to encode custom varying data type")
	errEncodeCustomPrimitiveFailed       = errors.New("failed to encode custom primitive")
	errEncodeNilFailed                   = errors.New("failed to encode nil")
	errEncodeOptionFailed                = errors.New("failed to encode option")
	errEncodeStructFailed                = errors.New("failed to encode struct")
	errEncodeArrayFailed                 = errors.New("failed to encode array")
	errEncodeSliceFailed                 = errors.New("failed to encode slice")
	errEncodeLengthFailed                = errors.New("failed to encode length")

	ErrResultAlreadySet                = errors.New("result already has an assigned value")
	ErrUnableToAddVaryingDataTypeValue = errors.New("unable to add VaryingDataTypeValue")
)
