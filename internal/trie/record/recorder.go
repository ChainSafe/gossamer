// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package record

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

// GetNodes returns all the nodes recorded.
// Note it does not copy its slice of nodes.
// It's fine to not copy them since the recorder
// is not used again after a call to GetNodes()
func (r *Recorder) GetNodes() (nodes []Node) {
	return r.nodes
}
