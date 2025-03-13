// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package saturating

import (
	"reflect"
	"unsafe"

	"golang.org/x/exp/constraints"
)

func getMaxUnsigned[T constraints.Integer]() (max T) {
	return ^max
}

// should only be called with signed integers
func getMinMaxSigned[T constraints.Integer]() (min T, max T) {
	sizeOf := (unsafe.Sizeof(min) * 8) - 1
	max = 1<<sizeOf - 1
	min = ^max
	return
}

// Add is saturating addition. Compute a + b, saturating at the numeric bounds instead of overflowing.
func Add[T constraints.Integer](a T, b T) T {
	aType := reflect.TypeOf(a)
	aKind := aType.Kind()
	switch aKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		aMin, aMax := getMinMaxSigned[T]()
		if b > 0 {
			if a > aMax-b {
				return aMax
			}
			return a + b
		} else {
			if a < aMin-b {
				return aMin
			}
			return a + b
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if b >= getMaxUnsigned[T]()-a {
			return getMaxUnsigned[T]()
		} else {
			return a + T(b)
		}
	default:
		panic("unreachable")
	}
}

// Mul is saturating multiply. Compute a * b, saturating at the numeric bounds instead of overflowing.
func Mul[T constraints.Integer](a T, b T) T {
	if a == 0 || b == 0 {
		return 0
	}
	if a == 1 {
		return b
	}
	if b == 1 {
		return a
	}

	aType := reflect.TypeOf(a)
	aKind := aType.Kind()
	switch aKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// a is signed, b is signed
		aMin, aMax := getMinMaxSigned[T]()
		maxMultiple := aMax / a
		minMultiple := aMin / a

		if minMultiple < maxMultiple {
			if b <= maxMultiple && b >= minMultiple {
				return a * b
			}
		} else {
			if b >= maxMultiple && b <= minMultiple {
				return a * b
			}
		}

		if a < 0 && b >= 0 || a >= 0 && b < 0 {
			return aMin
		}
		return aMax
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// a is unsigned, b is unsigned
		maxA := getMaxUnsigned[T]()
		maxMultiple := maxA / a

		if maxMultiple > b {
			return a * b
		}
		return maxA
	default:
		panic("unreachable")
	}
}

// Sub is saturating subtraction. Compute a - b, saturating at the numeric bounds instead of overflowing.
func Sub[T constraints.Unsigned](a T, b T) T {
	if b <= a {
		return a - b
	} else {
		return 0
	}

}

// Into will determine if the source value is too big to fit into the destination type then it will saturate the
// destination. T and U both have to signed, or both have to be unsigned. Function will panic if that condition is
// not met.
func Into[T, U constraints.Integer](n T) (dst U) {
	if n == 0 {
		return 0
	}

	nKind := reflect.TypeOf(n).Kind()
	dstKind := reflect.TypeOf(dst).Kind()

	switch nKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch dstKind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// n is signed, dst is signed
			sizeDst := unsafe.Sizeof(dst)
			sizeN := unsafe.Sizeof(n)
			dstMin, dstMax := getMinMaxSigned[U]()
			if n > 0 {
				if sizeDst < sizeN {
					if n >= T(dstMax) {
						return dstMax
					}
					return U(n)
				}
				// sizeDst >= sizeN
				if U(n) >= dstMax {
					return dstMax
				}
				return U(n)

			}
			// n < 0
			if sizeDst < sizeN {
				if n <= T(dstMin) {
					return dstMin
				}
				return U(n)
			}
			// sizeDst >= sizeN
			if U(n) <= dstMin {
				return dstMin
			}
			return U(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			panic("received unsigned integer for destination")
		default:
			panic("unreachable")
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch dstKind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			panic("received signed integer for destination")
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// n is unsigned, dst is unsigned
			sizeDst := unsafe.Sizeof(dst)
			sizeN := unsafe.Sizeof(n)

			switch {
			case sizeDst < sizeN:
				if n > T(getMaxUnsigned[U]()) {
					return getMaxUnsigned[U]()
				} else {
					return U(n)
				}
			default:
				return U(n)
			}
		default:
			panic("unreachable")
		}
	default:
		panic("unreachable")
	}
}
