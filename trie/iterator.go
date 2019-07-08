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
	"fmt"
	log "github.com/inconshreveable/log15"
)

// Entries returns all the key-value pairs in the trie as a map of keys to values
func (t *Trie) Entries() map[string][]byte {
	return t.entries(t.root, nil, make(map[string][]byte))
}

func (t *Trie) entries(current node, prefix []byte, kv map[string][]byte) map[string][]byte {
	switch c := current.(type) {
	case *branch:
		if c.value != nil {
			kv[string(nibblesToKeyLE(append(prefix, c.key...)))] = c.value
		}
		for i, child := range c.children {
			//t.entries(child, append(append(prefix, byte(i)), c.key...), kv)
			t.entries(child, append(prefix, append(c.key, byte(i))...), kv)
		}
	case *leaf:
		kv[string(nibblesToKeyLE(append(prefix, c.key...)))] = c.value
		return kv
	}

	return kv
}

// Print prints the trie through pre-order traversal
func (t *Trie) Print() {
	fmt.Println("printing trie...")
	t.print(t.root, nil, false)
}

func (t *Trie) PrintEncoding() {
	t.print(t.root, nil, true)
}

func (t *Trie) print(current node, prefix []byte, withEncoding bool) {
	h, err := NewHasher()
	if err != nil {
		log.Error("newHasher", "error", err)
	}
	var encoding []byte
	var hash []byte
	if withEncoding && current != nil {
		encoding, err = current.Encode()
		if err != nil {
			log.Error("encoding", "error", err)
		}
		hash, err = h.Hash(current)
		if err != nil {
			log.Error("hash", "error", err)
		}
	}

	switch c := current.(type) {
	case *branch:
		log.Info("branch", "key", fmt.Sprintf("%x", nibblesToKeyLE(append(prefix, c.key...))), "children", fmt.Sprintf("%b", c.childrenBitmap()), "value", fmt.Sprintf("%x", c.value))
		if withEncoding {
			log.Info("branch encoding ")
			printHexBytes(encoding)
			fmt.Printf("branch hash ")
			printHexBytes(hash)
		}
		for i, child := range c.children {
			t.print(child, append(append(prefix, byte(i)), c.key...), withEncoding)
		}
	case *leaf:
		fmt.Printf("leaf key %x value %x\n", nibblesToKeyLE(append(prefix, c.key...)), c.value)
		if withEncoding {
			fmt.Printf("leaf encoding ")
			printHexBytes(encoding)
			fmt.Printf("leaf hash ")
			printHexBytes(hash)
		}
	default:
		// do nothing
	}
}

func printHexBytes(in []byte) {
	fmt.Print("[")
	for i, b := range in {
		if i < len(in)-1 {
			fmt.Printf("%x, ", b)
		} else {
			fmt.Printf("%x", b)
		}
	}
	fmt.Println("]")
}
