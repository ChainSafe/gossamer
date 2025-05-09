// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

// Something that can fetch the runtime :code.
type FetchRuntimeCode interface {
	FetchRuntimeCode() []byte
}

// Wrapper to use a []byte as [FetchRuntimeCode].
type WrappedRuntimeCode struct {
	code []byte
}

func NewWrappedRuntimeCode(code []byte) *WrappedRuntimeCode {
	return &WrappedRuntimeCode{
		code: code,
	}
}

// Fetch the runtime :code.
// If the :code could not be found/not available, nil will be returned.
func (w *WrappedRuntimeCode) FetchRuntimeCode() []byte {
	return w.code
}

// The Wasm code of a Substrate runtime.
type RuntimeCode struct {
	// The code fetcher that can be used to lazily fetch the code.
	CodeFetcher FetchRuntimeCode
	// The optional heap pages this `code` should be executed with.
	// If `None` are given, the default value of the executor will be used.
	HeapPages *uint64
	// The hash of `code`.
	// The hashing algorithm isn't that important, as long as all runtime
	// code instances use the same.
	Hash []byte
}
