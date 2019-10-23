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
	"sync"

	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/polkadb"
)

// StateDB is a wrapper around a polkadb
type StateDB struct {
	Db     polkadb.Database
	Batch  polkadb.Batch
	Lock   sync.RWMutex
	Hasher *Hasher
}

// EncodeForDB traverses the trie recursively, encodes each node then SCALE encodes the encoding
// and appends them all together
func (t *Trie) EncodeForDB() ([]byte, error) {
	return t.encodeForDB(t.root, []byte{})
}

func (t *Trie) encodeForDB(n node, enc []byte) ([]byte, error) {
	nenc, err := n.Encode()
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
				enc, err = t.encodeForDB(child, enc)
				if err != nil {
					return enc, err
				}
			}
		}
	}

	return enc, nil
}

func (t *Trie) DecodeFromDB(enc []byte) error {
	r := &bytes.Buffer{}
	_, err := r.Write(enc)
	if err != nil {
		return err
	}

	sd := &scale.Decoder{r}
	scroot, err := sd.Decode([]byte{})
	if err != nil {
		return err
	}

	n := &bytes.Buffer{}
	_, err = n.Write(scroot.([]byte))
	if err != nil {
		return err
	}

	root, err := Decode(n)
	if err != nil {
		return err
	}

	t.root = root
	return t.decodeFromDB(r, root)
}

func (t *Trie) decodeFromDB(r io.Reader, prev node) error {
	sd := &scale.Decoder{r}

	if b, ok := prev.(*branch); ok {
		for i, child := range b.children {
			if child != nil {
				// there's supposed to be a child here, decode the next node and place it
				scnode, err := sd.Decode([]byte{})
				if err != nil {
					return err
				}

				n := &bytes.Buffer{}
				_, err = n.Write(scnode.([]byte))
				if err != nil {
					return err
				}

				next, err := Decode(n)
				if err != nil {
					return fmt.Errorf("could not decode child at %d: %s", i, err)
				}

				b.children[i] = next
				err = t.decodeFromDB(r, next)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// StoreInDB encodes the entire trie and writes it to the DB
// The key to the DB entry is the root hash of the trie
func (t *Trie) StoreInDB() error {
	enc, err := t.EncodeForDB()
	if err != nil {
		return err
	}

	encroot, err := t.Encode()
	if err != nil {
		return err
	}

	return t.db.Db.Put(encroot, enc)
}

// LoadFromDB loads an encoded trie from the DB where the key is `root`
func (t *Trie) LoadFromDB(root []byte) error {
	enctrie, err := t.db.Db.Get(root)
	if err != nil {
		return err
	}

	return t.DecodeFromDB(enctrie)
}
