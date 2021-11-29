// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie/node"
)

var _ node.Node = (*Leaf)(nil)

// Leaf is a leaf in the trie.
type Leaf struct {
	Key        []byte // partial key
	Value      []byte
	Dirty      bool
	Hash       []byte
	Encoding   []byte
	encodingMu sync.RWMutex
	Generation uint64
	sync.RWMutex
}

func (l *Leaf) String() string {
	if len(l.Value) > 1024 {
		return fmt.Sprintf("leaf key=%x value (hashed)=%x dirty=%v", l.Key, common.MustBlake2bHash(l.Value), l.Dirty)
	}
	return fmt.Sprintf("leaf key=%x value=%v dirty=%v", l.Key, l.Value, l.Dirty)
}
