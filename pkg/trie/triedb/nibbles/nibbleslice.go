// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibbles

import "slices"

// NOTE: This is to facilitate easier truncation, and to reference odd number
// lengths omitting the last nibble of the last byte.
type NibbleSlice struct {
	inner []byte
	len   uint
}

// Construct a new [NibbleSlice].
func NewNibbleSlice() NibbleSlice {
	return NibbleSlice{
		inner: make([]byte, 0),
	}
}

// Construct a new [NibbleSlice] from [Nibbles].
func NewNibbleSliceFromNibbles(s Nibbles) NibbleSlice {
	v := NewNibbleSlice()
	for i := uint(0); i < s.Len(); i++ {
		v.Push(s.At(i))
	}
	return v
}

// Returns true if [NibbleSlice] has zero length.
func (n NibbleSlice) IsEmpty() bool {
	return n.len == 0
}

// Try to get the nibble at the given offset.
func (n NibbleSlice) At(idx uint) uint8 {
	ix := idx / NibblesPerByte
	pad := idx % NibblesPerByte
	return atLeft(uint8(pad), n.inner[ix]) //nolint:gosec
}

// Push a nibble onto the [NibbleSlice]. Ignores the high 4 bits.
func (n *NibbleSlice) Push(nibble uint8) {
	i := n.len % NibblesPerByte
	if i == 0 {
		n.inner = append(n.inner, PushAtLeft(0, nibble, 0))
	} else {
		output := n.inner[len(n.inner)-1]
		n.inner[len(n.inner)-1] = PushAtLeft(uint8(i), nibble, output) //nolint:gosec
	}
	n.len++
}

// Try to pop a nibble off the NibbleVec. Fails if len == 0.
func (n *NibbleSlice) Pop() *uint8 {
	if n.IsEmpty() {
		return nil
	}
	b := n.inner[len(n.inner)-1]
	n.inner = n.inner[:len(n.inner)-1]
	n.len -= 1
	iNew := n.len % NibblesPerByte
	if iNew != 0 {
		n.inner = append(n.inner, PadLeft(b))
	}
	popped := atLeft(uint8(iNew), b)
	return &popped
}

// Append a [Partial]. Can be slow (alignement of partial).
func (n *NibbleSlice) AppendPartial(p Partial) {
	if p.First == 1 {
		n.Push(atLeft(1, p.PaddedNibble))
	}
	pad := uint(len(n.inner))*NibblesPerByte - n.len
	if pad == 0 {
		n.inner = append(n.inner, p.Data...)
	} else {
		kend := uint(len(n.inner)) - 1
		if len(p.Data) > 0 {
			n.inner[kend] = PadLeft(n.inner[kend])
			n.inner[kend] |= p.Data[0] >> 4
			for i := 0; i < len(p.Data)-1; i++ {
				n.inner = append(n.inner, p.Data[i]<<4|p.Data[i+1]>>4)
			}
			n.inner = append(n.inner, p.Data[len(p.Data)-1]<<4)
		}
	}
	n.len += uint(len(p.Data)) * NibblesPerByte
}

// Utility function for chaining two optional appending
// of [NibbleSlice] and/or a byte.
// Can be slow.
func (n *NibbleSlice) AppendOptionalSliceAndNibble(oSlice *Nibbles, oIndex *uint8) uint {
	var res uint
	if oSlice != nil {
		n.AppendPartial(oSlice.RightPartial())
		res += oSlice.Len()
	}
	if oIndex != nil {
		n.Push(*oIndex)
		res += 1
	}
	return res
}

// Get Prefix representation of this [NibbleSlice].
func (n NibbleSlice) Prefix() Prefix {
	split := n.len / NibblesPerByte
	pos := uint8(n.len % NibblesPerByte) //nolint:gosec
	if pos == 0 {
		return Prefix{
			Key: n.inner[:split],
		}
	} else {
		padded := PadLeft(n.inner[split])
		return Prefix{
			Key:    n.inner[:split],
			Padded: &padded,
		}
	}
}

func (n *NibbleSlice) Clear() {
	n.inner = make([]byte, 0)
	n.len = 0
}

// Remove the last num nibbles in a faster way than popping num times.
func (n *NibbleSlice) DropLasts(num uint) {
	if num == 0 {
		return
	}
	if num >= n.len {
		n.Clear()
		return
	}
	end := n.len - num
	endIndex := end / NibblesPerByte
	if end%NibblesPerByte != 0 {
		endIndex++
	}
	for i := endIndex; i < uint(len(n.inner)); endIndex++ {
		n.inner = n.inner[:len(n.inner)-1]
	}
	n.len = end
	pos := n.len % NibblesPerByte
	if pos != 0 {
		kl := len(n.inner) - 1
		n.inner[kl] = PadLeft(n.inner[kl])
	}
}

func (n NibbleSlice) Clone() NibbleSlice {
	return NibbleSlice{
		inner: slices.Clone(n.inner),
		len:   n.len,
	}
}
