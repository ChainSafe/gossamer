// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

type Action interface {
	isInsertAction()
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

func (Replace) isInsertAction() {}
func (r Replace) getNode() Node { return r.node }
func (Restore) isInsertAction() {}
func (r Restore) getNode() Node { return r.node }
func (Delete) isInsertAction()  {}
func (Delete) getNode() Node    { return nil }
