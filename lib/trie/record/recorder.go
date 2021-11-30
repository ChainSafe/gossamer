// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package record

import "errors"

var (
	ErrNoNextNode = errors.New("no next node")
)

// Recorder records the list of nodes found by Lookup.Find
type Recorder struct {
	nodes []Node
}

// NewRecorder creates a new recorder.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// Record appends a node to the list of visited nodes.
func (r *Recorder) Record(hash, rawData []byte) {
	r.nodes = append(r.nodes, Node{RawData: rawData, Hash: hash})
}

// Next returns the first node in the recorded list
// and removes it (shift operation).
func (r *Recorder) Next() (node Node, err error) {
	if len(r.nodes) == 0 {
		return node, ErrNoNextNode
	}

	node = r.nodes[0]
	r.nodes = r.nodes[1:]

	return node, nil
}

// IsEmpty returns true if no node is in in the current
// recorded list of nodes.
func (r *Recorder) IsEmpty() (empty bool) {
	return len(r.nodes) == 0
}
