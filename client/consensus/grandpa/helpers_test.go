package grandpa

import (
	"golang.org/x/exp/constraints"
)

// //// Fulfils Header interface ////
type testHeader[Hash constraints.Ordered, N constraints.Unsigned] struct {
	ParentHashField Hash
	NumberField     N
	StateRoot       Hash
	ExtrinsicsRoot  Hash
	HashField       Hash
}

func (s testHeader[Hash, N]) ParentHash() Hash {
	return s.ParentHashField
}

func (s testHeader[Hash, N]) Hash() Hash {
	return s.HashField
}

func (s testHeader[Hash, N]) Number() N {
	return s.NumberField
}

// //// Fulfils HeaderBackend interface //////
type testHeaderBackend[Hash constraints.Ordered, N constraints.Unsigned, H testHeader[Hash, N]] struct {
	header *testHeader[Hash, N]
}

func (backend testHeaderBackend[Hash, N, H]) Header(hash Hash) (*testHeader[Hash, N], error) {
	return backend.header, nil
}

func (backend testHeaderBackend[Hash, N, H]) Info() Info[N] {
	panic("unimplemented")
}

func (backend testHeaderBackend[Hash, N, H]) ExpectBlockHashFromID(id N) (Hash, error) {
	panic("unimplemented")
}

func (backend testHeaderBackend[Hash, N, H]) ExpectHeader(hash Hash) (H, error) {
	panic("unimplemented")
}
