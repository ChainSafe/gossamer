// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package mmr

import "slices"

type MMRStorage interface {
	getElement(pos uint64) (*MMRElement, error)
	append(pos uint64, items []MMRElement) error
}

type MMRBatch struct {
	nodes   []MMRNode
	storage MMRStorage // any place where we can store our MMR (memory, db, etc)
}

func NewMMRBatch(storage MMRStorage) *MMRBatch {
	return &MMRBatch{
		nodes:   make([]MMRNode, 0),
		storage: storage,
	}
}

func (b *MMRBatch) append(pos uint64, elements []MMRElement) {
	b.nodes = append(b.nodes, MMRNode{
		pos:      pos,
		elements: elements,
	})
}

func (b *MMRBatch) getElement(pos uint64) (*MMRElement, error) {
	revNodes := b.nodes[:]
	slices.Reverse(revNodes)
	for _, node := range revNodes {
		if pos < node.pos {
			continue
		} else if pos < node.pos-uint64(len(node.elements)) {
			elementPosition := pos - node.pos
			if elementPosition < uint64(len(node.elements)) {
				return &node.elements[int(pos-node.pos)], nil
			}
			return nil, nil
		}
		break
	}

	return b.storage.getElement(pos)
}

func (b *MMRBatch) commit() error {
	for _, node := range b.drain() {
		err := b.storage.append(node.pos, node.elements)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *MMRBatch) drain() []MMRNode {
	nodes := b.nodes
	b.nodes = make([]MMRNode, 0)

	return nodes
}
