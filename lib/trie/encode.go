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
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/scale"
)

// Encode traverses the trie recursively, encodes each node, SCALE encodes the encoded node, and appends them all together
func (t *Trie) Encode() ([]byte, error) {
	return encodeRecursive(t.root, []byte{})
}

func encodeRecursive(n node, enc []byte) ([]byte, error) {
	if n == nil {
		return []byte{}, nil
	}

	nenc, err := n.encode()
	if err != nil {
		return enc, err
	}

	scnenc, err := scale.Encode(nenc)
	if err != nil {
		return nil, err
	}

	enc = append(enc, scnenc...)

	switch n := n.(type) {
	case *branch:
		for _, child := range n.children {
			if child != nil {
				enc, err = encodeRecursive(child, enc)
				if err != nil {
					return enc, err
				}
			}
		}
	}

	return enc, nil
}

// Decode decodes a trie from the DB and sets the receiver to it
// The encoded trie must have been encoded with t.Encode
func (t *Trie) Decode(enc []byte) error {
	if bytes.Equal(enc, []byte{}) {
		return nil
	}

	r := &bytes.Buffer{}
	_, err := r.Write(enc)
	if err != nil {
		return err
	}

	sd := &scale.Decoder{Reader: r}
	scroot, err := sd.Decode([]byte{})
	if err != nil {
		return err
	}

	n := &bytes.Buffer{}
	_, err = n.Write(scroot.([]byte))
	if err != nil {
		return err
	}

	t.root, err = decode(n)
	if err != nil {
		return err
	}

	return decodeRecursive(r, t.root)
}

func decodeRecursive(r io.Reader, prev node) error {
	sd := &scale.Decoder{Reader: r}

	if b, ok := prev.(*branch); ok {
		for i, child := range b.children {
			if child != nil {
				// there's supposed to be a child here, decode the next node and place it
				// when we decode a branch node, we only know if a child is supposed to exist at a certain index (due to the
				// bitmap). we also have the hashes of the children, but we can't reconstruct the children from that. so
				// instead, we put an empty leaf node where the child should be, so when we reconstruct it in this function,
				// we can see that it's non-nil and we should decode the next node from the reader and place it here
				scnode, err := sd.Decode([]byte{})
				if err != nil {
					return err
				}

				n := &bytes.Buffer{}
				_, err = n.Write(scnode.([]byte))
				if err != nil {
					return err
				}

				b.children[i], err = decode(n)
				if err != nil {
					return fmt.Errorf("could not decode child at %d: %s", i, err)
				}

				err = decodeRecursive(r, b.children[i])
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
