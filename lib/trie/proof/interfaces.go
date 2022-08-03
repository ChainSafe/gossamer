package proof

type Metrics interface {
	NodesAdded(n uint32)
	NodesDeleted(n uint32)
}
