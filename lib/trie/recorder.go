package trie

// nodeRecord represets a record of a visited node
type nodeRecord struct {
	rawData []byte
	hash    []byte
}

// Recorder keeps the list of nodes find by Lookup.Find
type recorder []nodeRecord

// Record insert a node insede the recorded list
func (r *recorder) record(h, rd []byte) {
	*r = append(*r, nodeRecord{rawData: rd, hash: h})
}

// Next returns the current item the cursor is on and increment the cursor by 1
func (r *recorder) next() *nodeRecord {
	if !r.isEmpty() {
		n := (*r)[0]
		*r = (*r)[1:]
		return &n
	}

	return nil
}

// IsEmpty returns bool if there is data inside the slice
func (r *recorder) isEmpty() bool {
	return len(*r) <= 0
}
