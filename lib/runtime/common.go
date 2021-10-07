// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package runtime

// Int64ToPointerAndSize converts an int64 into a int32 pointer and a int32 length
func Int64ToPointerAndSize(in int64) (ptr, length int32) {
	return int32(in), int32(in >> 32)
}

// PointerAndSizeToInt64 converts int32 pointer and size to a int64
func PointerAndSizeToInt64(ptr, size int32) int64 {
	return int64(ptr) | (int64(size) << 32)
}
