package node

// Stats contains statistical information for a node.
type Stats struct {
	// Descendants is the number of descendant nodes for
	// this particular node.
	Descendants uint32
}

// NewStats creates a new Stats structure given the arguments.
func NewStats(descendants uint32) Stats {
	return Stats{
		Descendants: descendants,
	}
}

// GetDescendants returns the number of descendants in the branch.
func (b *Branch) GetDescendants() (descendants uint32) {
	return b.Stats.Descendants
}

// AddDescendants adds descendant nodes count to the node stats.
func (b *Branch) AddDescendants(n uint32) {
	b.Stats.Descendants += n
}

// SubDescendants subtracts descendant nodes count from the node stats.
func (b *Branch) SubDescendants(n uint32) {
	b.Stats.Descendants -= n
}
