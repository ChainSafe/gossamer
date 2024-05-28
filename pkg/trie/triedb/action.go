// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

// action is an enum used in the trie db to represent the different types of
// actions that can be performed during a trie insertion / deletion
// this is useful to perform this changes in our temporal structure
// see `Triedb.inspect` for more details
type action interface {
	getNode() Node
}

type (
	replace struct {
		node Node
	}
	restore struct {
		node Node
	}
	delete struct{}
)

func (r replace) getNode() Node { return r.node }
func (r restore) getNode() Node { return r.node }
func (delete) getNode() Node    { return nil }
