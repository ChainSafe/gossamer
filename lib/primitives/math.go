// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package primitives

import (
	"fmt"
	"unsafe"

	"golang.org/x/exp/constraints"
)

// SaturatingAdd computes a + b saturating at the numeric bounds instead of overflowing
func SaturatingAdd[T constraints.Integer](a, b T) T {
	switch any(a).(type) {
	case int, int8, int16, int32, int64:
		sizeOf := (unsafe.Sizeof(a) * 8) - 1

		var (
			maxValueOfSignedType T = 1<<sizeOf - 1
			minValueOfSignedType T = ^maxValueOfSignedType
		)

		return saturatingAddSigned(a, b, maxValueOfSignedType, minValueOfSignedType)
	case uint, uint8, uint16, uint32, uint64, uintptr:
		// the operation ^T(0) gives us the max value of type T
		// eg. if T is uint8 then it gives us 255
		return saturatingAddUnsigned(a, b, ^T(0))
	}

	panic(fmt.Sprintf("type %T not supported while performing SaturatingAdd", a))
}

func saturatingAddSigned[T constraints.Integer](a, b, max, min T) T {
	if b > 0 && a > max-b {
		return max
	}

	if b < 0 && a < min-b {
		return min
	}

	return a + b
}

func saturatingAddUnsigned[T constraints.Integer](a, b, max T) T {
	if a > max-b {
		return max
	}
	return a + b
}

// SaturatingSub computes a - b saturating at the numeric bounds instead of overflowing
func SaturatingSub[T constraints.Integer](a, b T) T {
	switch any(a).(type) {
	case int, int8, int16, int32, int64:
		sizeOf := (unsafe.Sizeof(a) * 8) - 1

		var (
			maxValueOfSignedType T = 1<<sizeOf - 1
			minValueOfSignedType T = ^maxValueOfSignedType
		)

		return saturatingSubSigned(a, b, maxValueOfSignedType, minValueOfSignedType)
	case uint, uint8, uint16, uint32, uint64, uintptr:
		// the operation ^T(0) gives us the max value of type T
		// eg. if T is uint8 then it gives us 255
		return saturatingSubUnsigned(a, b)
	}

	panic(fmt.Sprintf("type %T not supported while performing SaturatingSub", a))
}

func saturatingSubSigned[T constraints.Integer](a, b, max, min T) T {
	if b < 0 && a > max+b {
		return max
	}

	if b > 0 && a < min+b {
		return min
	}

	return a - b
}

func saturatingSubUnsigned[T constraints.Integer](a, b T) T {
	if a > b {
		return a - b
	}
	return 0
}
