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

	//fmt.Printf("node %v enc %x len %d\n", n, scnenc, len(scnenc))

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

// if prev node is branch, and not
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

// WriteToDB writes the trie to the underlying database batch writer
// Stores the merkle value of the node as the key and the encoded node as the value
// This does not actually write to the db, just to the batch writer
// Commit must be called afterwards to finish writing to the db
func (t *Trie) WriteToDB() error {
	t.db.Batch = t.db.Db.NewBatch()
	return t.writeToDB(t.root)
}

// writeToDB recursively attempts to write each node in the trie to the db batch writer
func (t *Trie) writeToDB(n node) error {
	_, err := t.writeNodeToDB(n)
	if err != nil {
		return err
	}

	switch n := n.(type) {
	case *branch:
		for _, child := range n.children {
			if child != nil {
				err = t.writeToDB(child)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// writeNodeToDB returns true if node is written to db batch writer, false otherwise
// if node is clean, it will not attempt to be written to the db
// otherwise if it's dirty, try to write it to db
func (t *Trie) writeNodeToDB(n node) (bool, error) {
	if !n.isDirty() {
		return false, nil
	}

	encRoot, err := Encode(n)
	if err != nil {
		return false, err
	}

	hash, err := t.db.Hasher.Hash(n)
	if err != nil {
		return false, err
	}

	t.db.Lock.Lock()
	err = t.db.Batch.Put(hash[:], encRoot)
	t.db.Lock.Unlock()

	n.setDirty(false)
	return true, err
}

// Commit writes the contents of the db's batch writer to the db
func (t *Trie) Commit() error {
	return t.db.Batch.Write()
}
