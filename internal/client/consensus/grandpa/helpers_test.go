// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// // //// Fulfils Header interface ////
// type testHeader[Hash constraints.Ordered, N constraints.Unsigned] struct {
// 	ParentHashField Hash
// 	NumberField     N
// 	StateRoot       Hash
// 	ExtrinsicsRoot  Hash
// 	HashField       Hash
// }

// func (s testHeader[Hash, N]) ParentHash() Hash {
// 	return s.ParentHashField
// }

// func (s testHeader[Hash, N]) Hash() Hash {
// 	return s.HashField
// }

// func (s testHeader[Hash, N]) Number() N {
// 	return s.NumberField
// }

// // //// Fulfils HeaderBackend interface //////
// type testHeaderBackend[Hash constraints.Ordered, N constraints.Unsigned] struct {
// 	header *Header[Hash, N]
// }

// func (backend testHeaderBackend[Hash, N]) Header(hash Hash) (*Header[Hash, N], error) {
// 	return backend.header, nil
// }

// func (backend testHeaderBackend[Hash, N]) Info() Info[N] {
// 	panic("unimplemented")
// }

// func (backend testHeaderBackend[Hash, N]) ExpectBlockHashFromID(id N) (Hash, error) {
// 	panic("unimplemented")
// }

// func (backend testHeaderBackend[Hash, N]) ExpectHeader(hash Hash) (Header[Hash, N], error) {
// 	panic("unimplemented")
// }
