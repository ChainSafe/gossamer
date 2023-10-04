// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package primitives

import (
	"fmt"
	"unsafe"

	"golang.org/x/exp/constraints"
)

// saturatingOperations applies the correct operation
// given the input types
func saturatingOperations[T constraints.Integer](a, b T,
	signedSaturatingOperation func(T, T, T, T) T,
	unsignedSaturatingOperation func(T, T) T,
) T {
	switch any(a).(type) {
	case int, int8, int16, int32, int64:
		// #nosec G103
		sizeOf := (unsafe.Sizeof(a) * 8) - 1

		var (
			maxValueOfSignedType T = 1<<sizeOf - 1
			minValueOfSignedType T = ^maxValueOfSignedType
		)

		return signedSaturatingOperation(a, b, maxValueOfSignedType, minValueOfSignedType)
	case uint, uint8, uint16, uint32, uint64, uintptr:
		// the operation ^T(0) gives us the max value of type T
		// eg. if T is uint8 then it gives us 255
		return unsignedSaturatingOperation(a, b)
	}

	panic(fmt.Sprintf("type %T not supported while performing SaturatingAdd", a))
}

// SaturatingAdd computes a + b saturating at the numeric bounds instead of overflowing
func SaturatingAdd[T constraints.Integer](a, b T) T {
	return saturatingOperations(a, b, saturatingAddSigned, saturatingAddUnsigned)
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

func saturatingAddUnsigned[T constraints.Integer](a, b T) T {
	// the operation ^T(0) gives us the max value of type T
	// eg. if T is uint8 then it gives us 255
	max := ^T(0)

	if a > max-b {
		return max
	}
	return a + b
}

// SaturatingSub computes a - b saturating at the numeric bounds instead of overflowing
func SaturatingSub[T constraints.Integer](a, b T) T {
	return saturatingOperations(a, b, saturatingSubSigned, saturatingSubUnsigned)
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
