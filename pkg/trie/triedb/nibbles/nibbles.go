// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibbles

import (
	"slices"
)

// Single nibble length in bit.
const BitsPerNibble uint8 = 4

// Number of nibble per byte.
const NibblesPerByte uint = 2

// Nibble (half a byte).
const PaddingBitmask uint8 = 0x0F

// Nibble-orientated view onto byte-slice, allowing nibble-precision offsets.
//
// This is meant to be an immutable struct. No operations actually change it.
type Nibbles struct {
	data   []uint8
	offset uint
}

// Construct new [Nibbles] from data and offset
func NewNibbles(data []byte, offset ...uint) Nibbles {
	var off uint
	if len(offset) > 0 {
		off = offset[0]
	}
	return Nibbles{
		data:   data,
		offset: off,
	}
}

// Construct new [Nibbles] from [NodeKey]
func NewNibblesFromNodeKey(from NodeKey) Nibbles {
	return NewNibbles(from.Data, from.Offset)
}

// Get the nibble at position i.
func (n Nibbles) At(i uint) uint8 {
	ix := (n.offset + i) / NibblesPerByte
	pad := uint8((n.offset + i) % NibblesPerByte) //nolint:gosec
	return atLeft(pad, n.data[ix])
}

func atLeft(ix, b uint8) uint8 {
	if ix == 1 {
		return b & 0x0F
	} else {
		return b >> BitsPerNibble
	}
}

// Return object which represents a view on to this slice (further) offset by i nibbles.
func (n Nibbles) Mid(i uint) Nibbles {
	return Nibbles{
		data:   slices.Clone(n.data),
		offset: n.offset + i,
	}
}

// Mask a byte, keeping left nibble.
func PadLeft(b uint8) uint8 {
	return b &^ PaddingBitmask
}

// Mask a byte, keeping right byte.
func PadRight(b uint8) uint8 {
	return b & PaddingBitmask
}

// A trie node prefix, it is the nibble path from the trie root
// to the trie node.
// For a node containing no partial key value it is the full key.
// For a value node or node containing a partial key, it is the full key minus its node partial
// nibbles (the node key can be split into prefix and node partial).
// Therefore it is always the leftmost portion of the node key, so its internal representation
// is a non expanded byte slice followed by a last padded byte representation.
// The padded byte is an optional padded value.
type Prefix struct {
	Key    []byte
	Padded *byte
}

func (p Prefix) JoinedBytes() []byte {
	if p.Padded != nil {
		return append(p.Key, *p.Padded)
	}
	return p.Key
}

// Return left portion of [Nibbles], if the slice
// originates from a full key it will be the Prefix of
// the node.
func (n Nibbles) Left() Prefix {
	split := n.offset / NibblesPerByte
	ix := uint8(n.offset % NibblesPerByte) //nolint:gosec
	if ix == 0 {
		return Prefix{Key: n.data[:split]}
	}
	padded := PadLeft(n.data[split])
	return Prefix{Key: slices.Clone(n.data[:split]), Padded: &padded}
}

func (n Nibbles) Len() uint {
	return uint(len(n.data))*NibblesPerByte - n.offset
}

// Advance the view on the slice by i nibbles.
func (n *Nibbles) Advance(i uint) {
	if n.Len() < i {
		panic("not enough nibbles to advance")
	}
	n.offset += i
}

// Move back to a previously valid fix offset position.
func (n Nibbles) Back(i uint) Nibbles {
	return Nibbles{data: n.data, offset: i}
}

func (n Nibbles) Equal(them Nibbles) bool {
	return n.Len() == them.Len() && n.StartsWith(them)
}

func (n Nibbles) StartsWith(them Nibbles) bool {
	return n.CommonPrefix(them) == them.Len()
}

// Calculate the number of common nibble between two left aligned bytes.
func leftCommon(a uint8, b uint8) uint {
	if a == b {
		return 2
	} else if PadLeft(a) == PadLeft(b) {
		return 1
	} else {
		return 0
	}
}

// Count the biggest common depth between two left aligned packed nibble slice.
func biggestDepth(v1 []uint8, v2 []uint8) uint {
	upperBound := len(v1)
	if len(v2) < upperBound {
		upperBound = len(v2)
	}
	for a := 0; a < upperBound; a++ {
		if v1[a] != v2[a] {
			return uint(a)*NibblesPerByte + leftCommon(v1[a], v2[a]) //nolint:gosec
		}
	}
	return uint(upperBound) * NibblesPerByte //nolint:gosec
}

// How many of the same nibbles at the beginning do we match with them?
func (n Nibbles) CommonPrefix(them Nibbles) uint {
	selfAlign := n.offset % NibblesPerByte
	themAlign := them.offset % NibblesPerByte
	if selfAlign == themAlign {
		selfStart := n.offset / NibblesPerByte
		themStart := them.offset / NibblesPerByte
		var first uint = 0
		if selfAlign != 0 {
			if PadRight(n.data[selfStart]) != PadRight(them.data[themStart]) {
				// warning only for radix 16
				return 0
			}
			selfStart += 1
			themStart += 1
			first += 1
		}
		return biggestDepth(n.data[selfStart:], them.data[themStart:]) + first
	} else {
		s := n.Len()
		if them.Len() < s {
			s = them.Len()
		}
		var i uint
		for i < s {
			if n.At(i) != them.At(i) {
				break
			}
			i++
		}
		return i
	}
}

// Helper function to create a [NodeKey].
func (n Nibbles) NodeKey() NodeKey {
	split := n.offset / NibblesPerByte
	offset := n.offset % NibblesPerByte
	return NodeKey{offset, n.data[split:]}
}

// Helper function to create a [NodeKey] for a given number of nibbles.
// Warning this method can be slow (number of nibble does not align the
// original padding).
func (n Nibbles) NodeKeyRange(nb uint) NodeKey {
	if nb >= n.Len() {
		return n.NodeKey()
	}
	if (n.offset+nb)%NibblesPerByte == 0 {
		// aligned
		start := n.offset / NibblesPerByte
		end := (n.offset + nb) / NibblesPerByte
		return NodeKey{
			Offset: n.offset % NibblesPerByte,
			Data:   n.data[start:end],
		}
	}
	// unaligned
	start := n.offset / NibblesPerByte
	end := (n.offset + nb) / NibblesPerByte
	ea := n.data[start : end+1]
	eaOffset := n.offset % NibblesPerByte
	nOffset := NumberPadding(nb)
	result := NodeKey{
		Offset: eaOffset,
		Data:   ea,
	}
	result.ShiftKey(nOffset)
	result.Data = result.Data[:len(result.Data)-1]
	return result
}

// Calculate the number of needed padding a array of nibble length i.
func NumberPadding(i uint) uint {
	return i % NibblesPerByte
}

// Representation of a nible slice (right aligned).
// It contains a right aligned padded first byte (first pair element is the number of nibbles
// (0 to max nb nibble - 1), second pair element is the padded nibble), and a slice over
// the remaining bytes.
type Partial struct {
	First        uint8
	PaddedNibble uint8
	Data         []byte
}

// Return [Partial] representation of this slice:
// first encoded byte and following slice.
func (n Nibbles) RightPartial() Partial {
	split := n.offset / NibblesPerByte
	nb := uint8(n.Len() % NibblesPerByte) //nolint:gosec
	if nb > 0 {
		return Partial{
			First:        nb,
			PaddedNibble: (n.data[split]),
			Data:         n.data[split+1:],
		}
	}
	return Partial{
		First:        0,
		PaddedNibble: 0,
		Data:         n.data[split:],
	}
}

// Return an iterator over [Partial] bytes representation.
func (n Nibbles) Right() []uint8 {
	p := n.RightPartial()
	var ret []uint8
	if p.First > 0 {
		ret = append(ret, PadRight(p.PaddedNibble))
	}
	for ix := 0; ix < len(p.Data); ix++ {
		ret = append(ret, p.Data[ix])
	}
	return ret
}

// Push uint8 nibble value at a given index into an existing byte.
func PushAtLeft(ix uint8, v uint8, into uint8) uint8 {
	var right uint8
	if ix == 1 {
		right = v
	} else {
		right = v << BitsPerNibble
	}
	return into | right
}

func (nb Nibbles) Clone() Nibbles {
	return Nibbles{
		data:   slices.Clone(nb.data),
		offset: nb.offset,
	}
}

// Get [Prefix] representation of the inner data.
//
// This means the entire inner data will be returned as [Prefix], ignoring any offset.
func (nb Nibbles) OriginalDataPrefix() Prefix {
	return Prefix{
		Key: nb.data,
	}
}

func (nb Nibbles) Compare(other Nibbles) int {
	s := nb.Len()
	if other.Len() < s {
		s = other.Len()
	}

	for i := uint(0); i < s; i++ {
		nbAt := nb.At(i)
		otherAt := other.At(i)
		if nbAt < otherAt {
			return -1
		} else if nbAt > otherAt {
			return 1
		}
	}

	if nb.Len() < other.Len() {
		return -1
	} else if nb.Len() > other.Len() {
		return 1
	} else {
		return 0
	}
}

// Partial node key type: offset and value.
// Offset is applied on first byte of array (bytes are right aligned).
type NodeKey struct {
	Offset uint
	Data   []byte
}

// Shifts right aligned key to add a given left offset.
// Resulting in possibly padding at both left and right
// (example usage when combining two keys).
func (nk *NodeKey) ShiftKey(offset uint) bool {
	oldOffset := nk.Offset
	nk.Offset = offset
	if oldOffset > offset {
		// shift left
		kl := len(nk.Data)
		for i := 0; i < kl-1; i++ {
			nk.Data[i] = nk.Data[i]<<4 | nk.Data[i+1]>>4
		}
		nk.Data[kl-1] = nk.Data[kl-1] << 4
		return true
	} else if oldOffset < offset {
		// shift right
		nk.Data = append(nk.Data, 0)
		for i := len(nk.Data) - 1; i >= 1; i-- {
			nk.Data[i] = nk.Data[i-1]<<4 | nk.Data[i]>>4
		}
		nk.Data[0] = nk.Data[0] >> 4
		return true
	}
	return false
}
