// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibble

import "github.com/ChainSafe/gossamer/pkg/trie/hashdb"

// NibbleSlice is a helper structure to store a slice of nibbles and a moving offset
// this is helpful to use it for example while we are looking for a key, we can define the full key in the data and
// moving the offset while we are going deep in the trie
type NibbleSlice struct {
	data   []byte
	offset uint
}

func NewNibbleSlice(data []byte) *NibbleSlice {
	return &NibbleSlice{data, 0}
}

func NewFromStored(i NodeKey) *NibbleSlice {
	return NewNibbleSliceWithPadding(i.nibbles.data, i.offset)
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

func (ns *NibbleSlice) StartsWith(other *NibbleSlice) bool {
	return ns.CommonPrefix(other) == other.Len()
}

func (ns *NibbleSlice) Eq(other *NibbleSlice) bool {
	return ns.Len() == other.Len() && ns.StartsWith(other)
}

func (ns *NibbleSlice) CommonPrefix(other *NibbleSlice) uint {
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
		if ns.At(i) != other.At(i) {
			break
		}
		i++
	}
	return i
}

func (ns *NibbleSlice) Left() hashdb.Prefix {
	split := ns.offset / NibblePerByte
	ix := (ns.offset % NibblePerByte)
	if ix == 0 {
		return hashdb.Prefix{
			PartialKey: ns.data[:split],
			PaddedByte: nil,
		}
	} else {
		padded := padRight(ns.data[split])

		return hashdb.Prefix{
			PartialKey: ns.data[:split],
			PaddedByte: &padded,
		}
	}
}

func (ns *NibbleSlice) OriginalDataAsPrefix() hashdb.Prefix {
	return hashdb.Prefix{
		PartialKey: ns.data,
		PaddedByte: nil,
	}
}
