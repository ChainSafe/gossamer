package trie

import "fmt"

// NodeRecord represets a record of a visited node
type NodeRecord struct {
	RawData []byte
	Hash    []byte
}

type Recorder []NodeRecord

func (r *Recorder) Record(h, rd []byte) {
	fmt.Printf("received ==> 0x%x\n", h)
	*r = append(*r, NodeRecord{
		RawData: rd,
		Hash:    h,
	})
}

func (r *Recorder) Len() int {
	return len(*r)
}

func (r *Recorder) Next() *NodeRecord {
	if r.Len() > 0 {
		n := (*r)[0]
		*r = (*r)[1:]
		return &n
	}

	return nil
}

func (r *Recorder) Peek() *NodeRecord {
	if r.Len() > 0 {
		return &(*r)[0]
	}
	return nil
}
