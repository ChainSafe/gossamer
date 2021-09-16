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

package types

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Block defines a state block
type Block struct {
	Header Header
	Body   Body
}

// NewBlock returns a new Block
func NewBlock(header Header, body Body) Block {
	return Block{
		Header: header,
		Body:   body,
	}
}

// NewEmptyBlock returns a new empty Block
func NewEmptyBlock() Block {
	return Block{
		Header: *NewEmptyHeader(),
		Body:   *NewBody(nil),
	}
}

// Empty returns a boolean indicating is the Block is empty
func (b *Block) Empty() bool {
	return b.Header.Empty() && len(b.Body) == 0
}

// Encode returns the SCALE encoding of a block
func (b *Block) Encode() ([]byte, error) {
	enc, err := scale.Marshal(b.Header)
	if err != nil {
		return nil, err
	}

	// block body is already SCALE encoded
	return append(enc, []byte(b.Body)...), nil
}

// MustEncode returns the SCALE encoded block and panics if it fails to encode
func (b *Block) MustEncode() []byte {
	enc, err := b.Encode()
	if err != nil {
		panic(err)
	}
	return enc
}

// DeepCopy returns a copy of the block
func (b *Block) DeepCopy() (Block, error) {
	bc := make([]byte, len(b.Body))
	copy(bc, b.Body)
	head, err := b.Header.DeepCopy()
	if err != nil {
		return Block{}, err
	}
	return Block{
		Header: *head,
		Body:   *NewBody(bc),
	}, nil
}
