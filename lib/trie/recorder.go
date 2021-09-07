package trie

import "github.com/ChainSafe/gossamer/lib/common"

// NodeRecord represets a record of a visited node
type NodeRecord struct {
	Depth   uint32
	RawData []byte
	Hash    common.Hash
}

// NodeRecorder records trie nodes as they pass it
type NodeRecorder struct {
	Nodes    []NodeRecord
	MinDepth uint32
}

// Record a visited node
func (r *NodeRecorder) Record(h common.Hash, rd []byte, depth uint32) {
	if depth >= r.MinDepth {
		r.Nodes = append(r.Nodes, NodeRecord{
			Depth:   depth,
			RawData: rd,
			Hash:    h,
		})
	}
}

// RecorderWithDepth create a NodeRecorder which only records nodes beyond a given depth
func NewRecorderWithDepth(d uint32) *NodeRecorder {
	return &NodeRecorder{
		MinDepth: d,
		Nodes:    []NodeRecord{},
	}
}

// NewRecoder create a NodeRecorder which records all given nodes
func NewRecoder() *NodeRecorder {
	return NewRecorderWithDepth(0)
}
