package runtime

type ProofMetrics interface {
	TrieMetrics
}

type RootHashMetrics interface {
	TrieMetrics
}

type TrieMetrics interface {
	NodesAdded(n uint32)
	NodesDeleted(n uint32)
}
