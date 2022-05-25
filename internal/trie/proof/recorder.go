// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

type recorder struct {
	nodes []visitedNode
}

type visitedNode struct {
	RawData []byte
	Hash    []byte
}

func newRecorder() *recorder {
	return &recorder{}
}

func (r *recorder) record(hash, rawData []byte) {
	r.nodes = append(r.nodes, visitedNode{RawData: rawData, Hash: hash})
}

// getNodes returns all the nodes recorded.
// Note it does not copy its slice of nodes.
// It's fine to not copy them since the recorder
// is not used again after a call to getNodes()
func (r *recorder) getNodes() (nodes []visitedNode) {
	return r.nodes
}
