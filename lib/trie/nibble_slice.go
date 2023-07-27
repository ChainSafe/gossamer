// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

const NibblePerByte uint = 2
const PaddingBitmask byte = 0x0F
const BitPerNibble = 4

type NibbleSlice struct {
	data   []byte
	offset uint
}

func NewNibbleSlice(data []byte) *NibbleSlice {
	return &NibbleSlice{data, 0}
}

func (ns *NibbleSlice) mid(i uint) *NibbleSlice {
	return &NibbleSlice{ns.data, ns.offset + i}
}

func (ns *NibbleSlice) left() *Prefix {
	split := ns.offset / NibblePerByte
	ix := uint8(ns.offset % NibblePerByte)
	if ix == 0 {
		return &Prefix{ns.data[:split], nil}
	}

	return &Prefix{ns.data[:split], padLeft(ns.data[split])}
}

func (ns *NibbleSlice) len() uint {
	return uint(len(ns.data))*NibblePerByte - ns.offset
}

func (ns *NibbleSlice) at(i uint) byte {
	ix := (ns.offset + i) / NibblePerByte
	pad := (ns.offset + i) % NibblePerByte
	b := ns.data[ix]
	if pad == 1 {
		return b & PaddingBitmask
	}
	return b >> BitPerNibble
}

func (ns *NibbleSlice) originalDataAsPrefix() Prefix {
	return Prefix{ns.data, nil}
}

func (ns *NibbleSlice) startsWith(other *NibbleSlice) bool {
	return ns.commonPrefix(other) == other.len()
}

func (ns *NibbleSlice) commonPrefix(other *NibbleSlice) uint {
	selfAlign := ns.offset % NibblePerByte
	otherAlign := other.offset % NibblePerByte
	if selfAlign == otherAlign {
		selfStart := ns.offset / NibblePerByte
		otherStart := other.offset / NibblePerByte
		first := uint(0)
		if selfAlign != 0 {
			if padRight(ns.data[selfStart]) != padRight(other.data[otherStart]) {
				return 0
			}
			selfStart++
			otherStart++
			first++
		}
		return biggestDepth(ns.data[selfStart:], other.data[otherStart:]) + first
	}

	s := minLength(ns.data, other.data)
	i := uint(0)
	for i < s {
		if ns.at(i) != other.at(i) {
			break
		}
		i++
	}
	return i
}

func padLeft(b byte) *byte {
	padded := (b & ^PaddingBitmask)
	return &padded
}

func padRight(b byte) *byte {
	padded := (b & PaddingBitmask)
	return &padded
}

// Count the biggest common depth between two left aligned packed nibble slice
func biggestDepth(v1, v2 []byte) uint {
	upperBound := minLength(v1, v2)

	for i := uint(0); i < upperBound; i++ {
		if v1[i] != v2[i] {
			return i*NibblePerByte + leftCommon(v1[i], v2[i])
		}
	}
	return upperBound * NibblePerByte
}

// Calculate the number of common nibble between two left aligned bytes
func leftCommon(a, b byte) uint {
	if a == b {
		return 2
	}
	if padLeft(a) == padLeft(b) {
		return 1
	} else {
		return 0
	}
}

func minLength(v1, v2 []byte) uint {
	if len(v1) < len(v2) {
		return uint(len(v1))
	}
	return uint(len(v2))
}
