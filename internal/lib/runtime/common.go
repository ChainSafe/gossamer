// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

// Int64ToPointerAndSize converts an int64 into a int32 pointer and a int32 length
func Int64ToPointerAndSize(in int64) (ptr, length int32) {
	return int32(in), int32(in >> 32)
}

// PointerAndSizeToInt64 converts int32 pointer and size to a int64
func PointerAndSizeToInt64(ptr, size int32) int64 {
	return int64(ptr) | (int64(size) << 32)
}
