package trie

// Metrics is the trie metrics interface.
type Metrics interface {
	NodesAdded(n uint32)
	NodesDeleted(n uint32)
}
