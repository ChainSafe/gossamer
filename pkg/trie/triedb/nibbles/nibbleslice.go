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

// Try to pop a nibble off the [NibbleSlice]. Fails if len == 0.
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

// Append another [NibbleSlice]. Can be slow (alignment of second slice).
func (n *NibbleSlice) Append(v NibbleSlice) {
	if v.len == 0 {
		return
	}

	finalLen := n.len + v.len
	offset := n.len % NibblesPerByte
	finalOffset := finalLen % NibblesPerByte
	lastIndex := n.len / NibblesPerByte
	if offset > 0 {
		n.inner[lastIndex] = PadLeft(n.inner[lastIndex]) | v.inner[0]>>4
		for i := uint(0); i < uint(len(v.inner))-1; i++ {
			n.inner = append(n.inner, v.inner[i]<<4|v.inner[i+1]>>4)
		}
		if finalOffset > 0 {
			n.inner = append(n.inner, v.inner[len(v.inner)-1]<<4)
		}
	} else {
		for i := uint(0); i < uint(len(v.inner)); i++ {
			n.inner = append(n.inner, v.inner[i])
		}
	}
	n.len += v.len
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

// Clones the underlying data in the returned [NibbleSlice].
func (n NibbleSlice) Clone() NibbleSlice {
	return NibbleSlice{
		inner: slices.Clone(n.inner),
		len:   n.len,
	}
}

// Length of the [NibbleSlice].
func (n NibbleSlice) Len() uint {
	return n.len
}

func (n NibbleSlice) asNibbles() *Nibbles {
	if n.len%NibblesPerByte == 0 {
		nibbles := NewNibbles(n.inner)
		return &nibbles
	}
	return nil
}

// Return an iterator over [Partial] bytes representation.
func (n NibbleSlice) Right() []byte {
	requirePadding := n.Len()%NibblesPerByte != 0
	var ix uint

	b := make([]byte, 0)
	for {
		if requirePadding && ix < uint(len(n.inner)) {
			if ix == 0 {
				ix++
				b = append(b, n.inner[ix-1]>>4)
			} else {
				ix++
				b = append(b, n.inner[ix-2]<<4|n.inner[ix-1]>>4)
			}
		} else if ix < uint(len(n.inner)) {
			ix++
			b = append(b, n.inner[ix-1])
		} else {
			break
		}
	}
	return b
}

// Returns a [NodeKey] representation
func (n NibbleSlice) NodeKey() NodeKey {
	if nibbles := n.asNibbles(); nibbles != nil {
		return nibbles.NodeKey()
	}
	return NodeKey{
		Offset: 1,
		Data:   n.Right(),
	}
}

// Returns the inner bytes
func (n NibbleSlice) Inner() []byte {
	return n.inner
}

func (n NibbleSlice) StartsWithNibbles(other Nibbles) bool {
	if n.Len() < other.Len() {
		return false
	}

	nAsNibbles := n.asNibbles()
	if nAsNibbles != nil {
		return nAsNibbles.StartsWith(other)
	}
	for i := uint(0); i < other.Len(); i++ {
		if n.At(i) != other.At(i) {
			return false
		}
	}
	return true
}
