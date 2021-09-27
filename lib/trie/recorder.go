package trie

// nodeRecord represets a record of a visited node
type nodeRecord struct {
	RawData []byte
	Hash    []byte
}

// Recorder keeps the list of nodes find by Lookup.Find
type recorder []nodeRecord

// Record insert a node insede the recorded list
func (r *recorder) record(h, rd []byte) {
	nr := nodeRecord{RawData: rd, Hash: h}
	*r = append(*r, nr)
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

// Peek returns the current item the cursor is on but dont increment the cursor by 1
func (r *recorder) peek() *nodeRecord {
	if !r.isEmpty() {
		return &(*r)[0]
	}
	return nil
}

// IsEmpty returns bool if there is data inside the slice
func (r *recorder) isEmpty() bool {
	return len(*r) > 0
}
