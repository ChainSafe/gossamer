// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

// Action is an enum used in the trie db to represent the different types of
// actions that can be performed during a trie insertion / deletion
// this is useful to perform this changes in our temporal structure
// see `Triedb.inspect` for more details
type Action interface {
	getNode() Node
}

type (
	Replace struct {
		node Node
	}
	Restore struct {
		node Node
	}
	Delete struct{}
)

func (r Replace) getNode() Node { return r.node }
func (r Restore) getNode() Node { return r.node }
func (Delete) getNode() Node    { return nil }
