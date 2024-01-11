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
	offset uint
}

func NewNibbleSlice(data []byte) *NibbleSlice {
	return &NibbleSlice{data, 0}
}

func NewNibbleSliceWithPadding(data []byte, padding uint) *NibbleSlice {
	return &NibbleSlice{data, padding}
}

func NewFromStored(i Prefix) *NibbleSlice {
	return &NibbleSlice{i.PartialKey, uint(*i.PaddedByte)}
}

func (ns *NibbleSlice) Clone() *NibbleSlice {
	data := make([]byte, len(ns.data))
	copy(data, ns.data)
	return &NibbleSlice{data, ns.offset}
}

func (ns *NibbleSlice) ToStored() NibbleSlice {
	split := ns.offset / NibblePerByte
	offset := ns.offset % NibblePerByte
	return NibbleSlice{
		data:   ns.data[split:],
		offset: offset,
	}
}

func (ns *NibbleSlice) ToStoredRange(nb uint) NibbleSlice {
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

func (ns *NibbleSlice) IsEmpty() bool {
	return len(ns.Data()) == 0
}

func (ns *NibbleSlice) Advance(i uint) {
	if ns.Len() < i {
		panic("Cannot advance more than the length of the slice")
	}
	ns.offset += i
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

func (ns *NibbleSlice) StartsWith(other NibbleSlice) bool {
	return ns.CommonPrefix(other) == other.Len()
}

func (ns *NibbleSlice) Eq(other NibbleSlice) bool {
	return ns.Len() == other.Len() && ns.StartsWith(other)
}

func (ns *NibbleSlice) CommonPrefix(other NibbleSlice) uint {
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

func (ns *NibbleSlice) Left() Prefix {
	split := ns.offset / NibblePerByte
	ix := (ns.offset % NibblePerByte)
	if ix == 0 {
		return Prefix{
			PartialKey: ns.data[:split],
			PaddedByte: nil,
		}
	} else {
		padded := padRight(ns.data[split])

		return Prefix{
			PartialKey: ns.data[:split],
			PaddedByte: &padded,
		}
	}
}

func (ns *NibbleSlice) OriginalDataAsPrefix() Prefix {
	return Prefix{
		PartialKey: ns.data,
		PaddedByte: nil,
	}
}

func CombineKeys(start *NibbleSlice, end NibbleSlice) {
	if start.offset >= NibblePerByte || end.offset >= NibblePerByte {
		panic("Cannot combine keys")
	}
	finalOffset := (start.offset + end.offset) % NibblePerByte
	ShiftKey(start, finalOffset)
	var st uint
	if end.offset > 0 {
		startLen := start.Len()
		start.data[startLen-1] = padRight(end.data[0])
		st = 1
	} else {
		st = 0
	}
	for i := st; i < end.Len(); i++ {
		start.data = append(start.data, end.data[i])
	}
}
