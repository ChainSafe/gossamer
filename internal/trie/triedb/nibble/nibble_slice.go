// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibble

import "github.com/ChainSafe/gossamer/internal/trie/hashdb"

type NibbleSlice struct {
	data   []byte
	offset uint
}

func NewNibbleSlice(data []byte) *NibbleSlice {
	return &NibbleSlice{data, 0}
}

func NewNibbleSliceWithPadding(data []byte, padding uint) *NibbleSlice {
	return &NibbleSlice{data, padding}
}

func (ns *NibbleSlice) Data() []byte {
	return ns.data
}

func (ns *NibbleSlice) Offset() uint {
	return ns.offset
}

func (ns *NibbleSlice) Mid(i uint) *NibbleSlice {
	return &NibbleSlice{ns.data, ns.offset + i}
}

func (ns *NibbleSlice) Left() *hashdb.Prefix {
	split := ns.offset / NibblePerByte
	ix := uint8(ns.offset % NibblePerByte)
	if ix == 0 {
		return &hashdb.Prefix{Data: ns.data[:split], Padded: nil}
	}

	padding := PadLeft(ns.data[split])

	return &hashdb.Prefix{Data: ns.data[:split], Padded: &padding}
}

func (ns *NibbleSlice) Len() uint {
	return uint(len(ns.data))*NibblePerByte - ns.offset
}

func (ns *NibbleSlice) At(i uint) byte {
	ix := (ns.offset + i) / NibblePerByte
	pad := (ns.offset + i) % NibblePerByte
	b := ns.data[ix]
	if pad == 1 {
		return b & PaddingBitmask
	}
	return b >> BitPerNibble
}

func (ns *NibbleSlice) OriginalDataAsPrefix() hashdb.Prefix {
	return hashdb.Prefix{Data: ns.data, Padded: nil}
}

func (ns *NibbleSlice) StartsWith(other *NibbleSlice) bool {
	return ns.commonPrefix(other) == other.Len()
}

func (ns *NibbleSlice) Eq(other *NibbleSlice) bool {
	return ns.Len() == other.Len() && ns.StartsWith(other)
}

func (ns *NibbleSlice) commonPrefix(other *NibbleSlice) uint {
	selfAlign := ns.offset % NibblePerByte
	otherAlign := other.offset % NibblePerByte
	if selfAlign == otherAlign {
		selfStart := ns.offset / NibblePerByte
		otherStart := other.offset / NibblePerByte
		first := uint(0)
		if selfAlign != 0 {
			if PadRight(ns.data[selfStart]) != PadRight(other.data[otherStart]) {
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
		if ns.At(i) != other.At(i) {
			break
		}
		i++
	}
	return i
}
