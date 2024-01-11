// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibble

// Nibble-orientated view onto byte-slice, allowing nibble-precision offsets.
//
// This is an immutable struct. No operations actually change it.
//
// # Example (TODO: translate to go)
// ```snippet
// use patricia_trie::nibbleslice::NibbleSlice;
//
//	fn main() {
//	  let d1 = &[0x01u8, 0x23, 0x45];
//	  let d2 = &[0x34u8, 0x50, 0x12];
//	  let d3 = &[0x00u8, 0x12];
//	  let n1 = NibbleSlice::new(d1);			// 0,1,2,3,4,5
//	  let n2 = NibbleSlice::new(d2);			// 3,4,5,0,1,2
//	  let n3 = NibbleSlice::new_offset(d3, 1);	// 0,1,2
//	  assert!(n1 > n3);							// 0,1,2,... > 0,1,2
//	  assert!(n1 < n2);							// 0,... < 3,...
//	  assert!(n2.mid(3) == n3);					// 0,1,2 == 0,1,2
//	  assert!(n1.starts_with(&n3));
//	  assert_eq!(n1.common_prefix(&n3), 3);
//	  assert_eq!(n2.mid(3).common_prefix(&n1), 3);
//	}
//
// ```
type NibbleSlice struct {
	data   []byte
	offset int
}

// NewNibbleSlice creates a new nibble slice from a byte slice
func NewNibbleSlice(data []byte) *NibbleSlice {
	return &NibbleSlice{data, 0}
}

// NewNibbleSliceWithOffset creates a new nibble slice from a byte slice with an offset
func NewNibbleSliceWithOffset(data []byte, offset int) *NibbleSlice {
	return &NibbleSlice{data, offset}
}

// Clone creates a deep copy of the nibble slice
func (ns *NibbleSlice) Clone() *NibbleSlice {
	data := make([]byte, len(ns.data))
	copy(data, ns.data)
	return &NibbleSlice{data, ns.offset}
}

// ToStored is a helper function to create a node key from this nibble slice
func (ns *NibbleSlice) ToStored() NibbleSlice {
	split := ns.offset / NibblePerByte
	offset := ns.offset % NibblePerByte
	return NibbleSlice{
		data:   ns.data[split:],
		offset: offset,
	}
}

// ToStoredRange is a helper function to create a node key from this `NibbleSlice` and for a given number of nibble
func (ns *NibbleSlice) ToStoredRange(nb int) NibbleSlice {
	if nb > ns.Len() {
		ns.ToStored()
	}
	if (ns.offset+nb)%NibblePerByte == 0 {
		// aligned
		start := ns.offset / NibblePerByte
		end := (ns.offset + nb) / NibblePerByte
		return NibbleSlice{
			data:   ns.data[start:end],
			offset: ns.offset % NibblePerByte,
		}
	} else {
		// unaligned
		start := ns.offset / NibblePerByte
		end := (ns.offset + nb) / NibblePerByte
		ea := ns.data[start : end+1]
		eaOffset := ns.offset
		nOffset := NumberPadding(nb)
		result := NibbleSlice{
			ea,
			eaOffset,
		}
		ShiftKey(&result, nOffset)
		result.data = result.data[:len(result.data)-1]
		return result
	}
}

// IsEmpty Return true if the slice contains no nibbles
func (ns *NibbleSlice) IsEmpty() bool {
	return ns.Len() == 0
}

// Advance the view on the slice by `i` nibbles
func (ns *NibbleSlice) Advance(i int) {
	if ns.Len() < i {
		panic("Cannot advance more than the length of the slice")
	}
	ns.offset += i
}

// Data returns the underlying byte slice
func (ns *NibbleSlice) Data() []byte {
	return ns.data
}

// Offset returns the offset of the nibble slice
func (ns *NibbleSlice) Offset() int {
	return ns.offset
}

// Mid returns a new nibble slice object which represents a view on to this slice (further) offset by `i` nibbles
func (ns *NibbleSlice) Mid(i int) *NibbleSlice {
	return &NibbleSlice{ns.data, ns.offset + i}
}

// Len returns the length of the nibble slice considering the offset
func (ns *NibbleSlice) Len() int {
	return len(ns.data)*NibblePerByte - ns.offset
}

// At returns the nibble at position `i`
func (ns *NibbleSlice) At(i int) byte {
	ix := (ns.offset + i) / NibblePerByte
	pad := (ns.offset + i) % NibblePerByte
	b := ns.data[ix]
	if pad == 1 {
		return b & PaddingBitmask
	}
	return b >> BitPerNibble
}

// StartsWith returns true if this nibble slice start with the same same nibbles contained in `other`
func (ns *NibbleSlice) StartsWith(other NibbleSlice) bool {
	return ns.CommonPrefix(other) == other.Len()
}

// Eq returns true if this nibble slice is equal to `other`
func (ns *NibbleSlice) Eq(other NibbleSlice) bool {
	return ns.Len() == other.Len() && ns.StartsWith(other)
}

// CommonPrefix return the amount of same nibbles at the beggining do we match with other
func (ns *NibbleSlice) CommonPrefix(other NibbleSlice) int {
	selfAlign := ns.offset % NibblePerByte
	otherAlign := other.offset % NibblePerByte
	if selfAlign == otherAlign {
		selfStart := ns.offset / NibblePerByte
		otherStart := other.offset / NibblePerByte
		first := 0
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
	i := 0
	for i < s {
		if ns.At(i) != other.At(i) {
			break
		}
		i++
	}
	return i
}

// Left returns left portion of `NibbleSlice`
// if the slice originates from a full key it will be the `Prefix of the node`.
func (ns *NibbleSlice) Left() Prefix {
	split := ns.offset / NibblePerByte
	ix := (ns.offset % NibblePerByte)
	prefix := Prefix{
		PartialKey: ns.data[:split],
		PaddedByte: nil,
	}
	if ix != 0 {
		padded := padRight(ns.data[split])
		prefix.PaddedByte = &padded
	}

	return prefix
}

// OriginalDataAsPrefix gets `Prefix` representation of the inner data.
// This means the entire inner data will be returned as `Prefix`, ignoring any `offset`.
func (ns *NibbleSlice) OriginalDataAsPrefix() Prefix {
	return Prefix{
		PartialKey: ns.data,
		PaddedByte: nil,
	}
}
