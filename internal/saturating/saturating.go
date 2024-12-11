package saturating

import (
	"math"
	"unsafe"

	"golang.org/x/exp/constraints"
)

func getMax(n uintptr) uint {
	switch n {
	case 1:
		return math.MaxUint8
	case 2:
		return math.MaxUint16
	case 4:
		return math.MaxUint32
	case 8:
		return math.MaxUint64
	default:
		panic("huh?")
	}
}

func Sub[T, U constraints.Unsigned](a T, b U) T {
	sizeA := unsafe.Sizeof(a)
	sizeB := unsafe.Sizeof(b)

	switch {
	case sizeB > sizeA:
		if uint(b) <= getMax(sizeA) {
			if T(b) > a {
				return 0
			} else {
				return a - T(b)
			}
		} else {
			return 0
		}
	default:
		// sizeb <= sizeA
		if T(b) <= a {
			return a - T(b)
		} else {
			return 0
		}
	}
}

func Into[T, U constraints.Unsigned](n T) (dst U) {
	sizeDst := unsafe.Sizeof(dst)
	sizeN := unsafe.Sizeof(n)

	switch {
	case sizeDst < sizeN:
		if uint(n) > getMax(sizeDst) {
			return U(getMax(sizeN))
		} else {
			return U(n)
		}
	default:
		return U(n)
	}
}
