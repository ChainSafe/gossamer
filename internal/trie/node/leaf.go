// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/qdm12/gotree"
)

var _ Node = (*Leaf)(nil)

// Leaf is a leaf in the trie.
type Leaf struct {
	// Partial key bytes in nibbles (0 to f in hexadecimal)
	Key   []byte
	Value []byte
	// Dirty is true when the branch differs
	// from the node stored in the database.
	Dirty      bool
	HashDigest []byte
	Encoding   []byte
	encodingMu sync.RWMutex
	// Generation is incremented on every trie Snapshot() call.
	// Each node also contain a certain Generation number,
	// which is updated to match the trie Generation once they are
	// inserted, moved or iterated over.
	Generation uint64
	sync.RWMutex
}

// NewLeaf creates a new leaf using the arguments given.
func NewLeaf(key, value []byte, dirty bool, generation uint64) *Leaf {
	return &Leaf{
		Key:        key,
		Value:      value,
		Dirty:      dirty,
		Generation: generation,
	}
}

// Type returns LeafType.
func (l *Leaf) Type() Type {
	return LeafType
}

func (l *Leaf) String() string {
	return l.StringNode().String()
}

// StringNode returns a gotree compatible node for String methods.
func (l *Leaf) StringNode() (stringNode *gotree.Node) {
	stringNode = gotree.New("Leaf")
	stringNode.Appendf("Generation: %d", l.Generation)
	stringNode.Appendf("Dirty: %t", l.Dirty)
	stringNode.Appendf("Key: " + bytesToString(l.Key))
	stringNode.Appendf("Value: " + bytesToString(l.Value))
	stringNode.Appendf("Calculated encoding: " + bytesToString(l.Encoding))
	stringNode.Appendf("Calculated digest: " + bytesToString(l.HashDigest))
	return stringNode
}

func bytesToString(b []byte) (s string) {
	switch {
	case b == nil:
		return "nil"
	case len(b) <= 20:
		return fmt.Sprintf("0x%x", b)
	default:
		return fmt.Sprintf("0x%x...%x", b[:8], b[len(b)-8:])
	}

}

type MockLeaf struct {
	Leaf

	Fail bool
}

// Encode is a hijackable Encode method.
func (m *MockLeaf) Encode(buffer Buffer) (err error) {
	if m.Fail {
		return errors.New("some error")
	}

	m.Leaf.encodingMu.RLock()
	if !m.Leaf.Dirty && m.Leaf.Encoding != nil {
		_, err = buffer.Write(m.Leaf.Encoding)
		m.Leaf.encodingMu.RUnlock()
		if err != nil {
			return fmt.Errorf("cannot write stored encoding to buffer: %w", err)
		}
		return nil
	}
	m.Leaf.encodingMu.RUnlock()

	err = m.Leaf.encodeHeader(buffer)
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	keyLE := codec.NibblesToKeyLE(m.Leaf.Key)
	_, err = buffer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	encodedValue, err := scale.Marshal(m.Leaf.Value) // TODO scale encoder to write to buffer
	if err != nil {
		return fmt.Errorf("cannot scale marshal value: %w", err)
	}

	_, err = buffer.Write(encodedValue)
	if err != nil {
		return fmt.Errorf("cannot write scale encoded value to buffer: %w", err)
	}

	// TODO remove this copying since it defeats the purpose of `buffer`
	// and the sync.Pool.
	m.Leaf.encodingMu.Lock()
	defer m.Leaf.encodingMu.Unlock()
	m.Leaf.Encoding = make([]byte, buffer.Len())
	copy(m.Leaf.Encoding, buffer.Bytes())
	return nil
}
