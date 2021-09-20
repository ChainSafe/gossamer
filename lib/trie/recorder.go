package trie

// NodeRecord represets a record of a visited node
type NodeRecord struct {
	RawData []byte
	Hash    []byte
}

// Recorder keeps the list of nodes find by Lookup.Find
type Recorder []NodeRecord

// Record insert a node insede the recorded list
func (r *Recorder) Record(h, rd []byte) {
	nr := NodeRecord{RawData: rd, Hash: h}
	*r = append(*r, nr)
}

// Len returns the current length of the recorded list
func (r *Recorder) Len() int {
	return len(*r)
}

// Next returns the current item the cursor is on and increment the cursor by 1
func (r *Recorder) Next() *NodeRecord {
	if r.Len() > 0 {
		n := (*r)[0]
		*r = (*r)[1:]
		return &n
	}

	return nil
}

// Peek returns the current item the cursor is on but dont increment the cursor by 1
func (r *Recorder) Peek() *NodeRecord {
	if r.Len() > 0 {
		return &(*r)[0]
	}
	return nil
}
