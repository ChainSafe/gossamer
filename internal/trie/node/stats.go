package node

// GetDescendants returns the number of descendants in the branch.
func (b *Branch) GetDescendants() (descendants uint32) {
	return b.Descendants
}

// AddDescendants adds descendant nodes count to the node stats.
func (b *Branch) AddDescendants(n uint32) {
	b.Descendants += n
}

// SubDescendants subtracts descendant nodes count from the node stats.
func (b *Branch) SubDescendants(n uint32) {
	b.Descendants -= n
}
