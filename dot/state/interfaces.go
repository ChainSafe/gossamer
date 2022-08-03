package state

type Metrics interface {
	TrieMetrics
}

type RootHashMetrics interface {
	TrieMetrics
}

type ProofMetrics interface {
	TrieMetrics
}

type TrieMetrics interface {
	NodesAdded(n uint32)
	NodesDeleted(n uint32)
}
