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

package trie

import (
	"bytes"
	"context"
	"hash"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
	"golang.org/x/sync/errgroup"

	"golang.org/x/crypto/blake2b"
)

type sliceBuffer []byte

func (b *sliceBuffer) Write(data []byte) (n int, err error) {
	*b = append(*b, data...)
	return len(data), nil
}

func (b *sliceBuffer) Reset() {
	*b = (*b)[:0]
}

// Hasher is a wrapper around a hash function
type Hasher struct {
	hash     hash.Hash
	tmp      sliceBuffer
	parallel bool // Whether to use parallel threads when hashing
}

// HasherPool creates a pool of Hasher.
var HasherPool = sync.Pool{
	New: func() interface{} {
		h, _ := blake2b.New256(nil)

		return &Hasher{
			tmp:  make(sliceBuffer, 0, 520), // cap is as large as a full branch node.
			hash: h,
		}
	},
}

// NewHasher create new Hasher instance
func NewHasher(parallel bool) *Hasher {
	h := HasherPool.Get().(*Hasher)
	h.parallel = parallel
	return h
}

func returnHasherToPool(h *Hasher) {
	h.tmp.Reset()
	h.hash.Reset()
	HasherPool.Put(h)
}

// Hash encodes the node and then hashes it if its encoded length is > 32 bytes
func (h *Hasher) Hash(n node) (res []byte, err error) {
	encNode, err := h.encode(n)
	if err != nil {
		return nil, err
	}

	// if length of encoded leaf is less than 32 bytes, do not hash
	if len(encNode) < 32 {
		return encNode, nil
	}

	h.hash.Reset()
	// otherwise, hash encoded node
	_, err = h.hash.Write(encNode)
	if err == nil {
		res = h.hash.Sum(nil)
	}

	return res, err
}

// encode is the high-level function wrapping the encoding for different node types
// encoding has the following format:
// NodeHeader | Extra partial key length | Partial Key | Value
func (h *Hasher) encode(n node) ([]byte, error) {
	switch n := n.(type) {
	case *branch:
		return h.encodeBranch(n)
	case *leaf:
		return h.encodeLeaf(n)
	case nil:
		return []byte{0}, nil
	}

	return nil, nil
}

// encodeBranch encodes a branch with the encoding specified at the top of this package
func (h *Hasher) encodeBranch(b *branch) ([]byte, error) {
	if !b.dirty && b.encoding != nil {
		return b.encoding, nil
	}
	h.tmp.Reset()

	encoding, err := b.header()
	h.tmp = append(h.tmp, encoding...)
	if err != nil {
		return nil, err
	}

	h.tmp = append(h.tmp, nibblesToKeyLE(b.key)...)
	h.tmp = append(h.tmp, common.Uint16ToBytes(b.childrenBitmap())...)

	if b.value != nil {
		buffer := bytes.Buffer{}
		se := scale.Encoder{Writer: &buffer}
		_, err = se.Encode(b.value)
		if err != nil {
			return h.tmp, err
		}
		h.tmp = append(h.tmp, buffer.Bytes()...)
	}

	if h.parallel {
		wg, _ := errgroup.WithContext(context.Background())
		resBuff := make([][]byte, 16)
		for i := 0; i < 16; i++ {
			func(i int) {
				wg.Go(func() error {
					child := b.children[i]
					if child == nil {
						return nil
					}

					hasher := NewHasher(false)
					defer returnHasherToPool(hasher)

					encChild, err := hasher.Hash(child)
					if err != nil {
						return err
					}

					scEncChild, err := scale.Encode(encChild)
					if err != nil {
						return err
					}
					resBuff[i] = scEncChild
					return nil
				})
			}(i)
		}
		if err := wg.Wait(); err != nil {
			return nil, err
		}

		for _, v := range resBuff {
			if v != nil {
				h.tmp = append(h.tmp, v...)
			}
		}
	} else {
		for i := 0; i < 16; i++ {
			if child := b.children[i]; child != nil {
				hasher := NewHasher(false)
				defer returnHasherToPool(hasher)

				encChild, err := hasher.Hash(child)
				if err != nil {
					return nil, err
				}

				scEncChild, err := scale.Encode(encChild)
				if err != nil {
					return nil, err
				}
				h.tmp = append(h.tmp, scEncChild...)
			}
		}
	}

	return h.tmp, nil
}

// encodeLeaf encodes a leaf with the encoding specified at the top of this package
func (h *Hasher) encodeLeaf(l *leaf) ([]byte, error) {
	if !l.dirty && l.encoding != nil {
		return l.encoding, nil
	}

	h.tmp.Reset()

	encoding, err := l.header()
	h.tmp = append(h.tmp, encoding...)
	if err != nil {
		return nil, err
	}

	h.tmp = append(h.tmp, nibblesToKeyLE(l.key)...)

	buffer := bytes.Buffer{}
	se := scale.Encoder{Writer: &buffer}

	_, err = se.Encode(l.value)
	if err != nil {
		return nil, err
	}

	h.tmp = append(h.tmp, buffer.Bytes()...)
	l.encoding = h.tmp
	return h.tmp, nil
}
