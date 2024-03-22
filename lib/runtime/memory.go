// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

// Memory is the interface for WASM memory
type Memory interface {
	// Size returns the size in bytes available. e.g. If the underlying memory
	// has 1 page: 65536
	//
	// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#-hrefsyntax-instr-memorymathsfmemorysize%E2%91%A0
	Size() uint64

	// Grow increases memory by the delta in pages (65536 bytes per page).
	// The return val is the previous memory size in pages, or false if the
	// delta was ignored as it exceeds MemoryDefinition.Max.
	//
	// # Notes
	//
	//   - This is the same as the "memory.grow" instruction defined in the
	//	  WebAssembly Core Specification, except returns false instead of -1.
	//   - When this returns true, any shared views via Read must be refreshed.
	//
	// See MemorySizer Read and https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#grow-mem
	Grow(deltaPages uint32) (previousPages uint32, ok bool)

	// ReadByte reads a single byte from the underlying buffer at the offset or returns false if out of range.
	ReadByte(offset uint32) (byte, bool) //nolint:govet

	// ReadUint64Le reads a uint64 in little-endian encoding from the underlying buffer at the offset or returns false
	// if out of range.
	ReadUint64Le(offset uint32) (uint64, bool)

	// WriteUint64Le writes the value in little-endian encoding to the underlying buffer at the offset in or returns
	// false if out of range.
	WriteUint64Le(offset uint32, v uint64) bool

	// Read reads byteCount bytes from the underlying buffer at the offset or
	// returns false if out of range.
	//
	// For example, to search for a NUL-terminated string:
	//	buf, _ = memory.Read(offset, byteCount)
	//	n := bytes.IndexByte(buf, 0)
	//	if n < 0 {
	//		// Not found!
	//	}
	//
	// Write-through
	//
	// This returns a view of the underlying memory, not a copy. This means any
	// writes to the slice returned are visible to Wasm, and any updates from
	// Wasm are visible reading the returned slice.
	//
	// For example:
	//	buf, _ = memory.Read(offset, byteCount)
	//	buf[1] = 'a' // writes through to memory, meaning Wasm code see 'a'.
	//
	// If you don't intend-write through, make a copy of the returned slice.
	//
	// When to refresh Read
	//
	// The returned slice disconnects on any capacity change. For example,
	// `buf = append(buf, 'a')` might result in a slice that is no longer
	// shared. The same exists Wasm side. For example, if Wasm changes its
	// memory capacity, ex via "memory.grow"), the host slice is no longer
	// shared. Those who need a stable view must set Wasm memory min=max, or
	// use wazero.RuntimeConfig WithMemoryCapacityPages to ensure max is always
	// allocated.
	Read(offset uint32, byteCount uint64) ([]byte, bool)

	// WriteByte writes a single byte to the underlying buffer at the offset in or returns false if out of range.
	WriteByte(offset uint32, v byte) bool //nolint:govet

	// Write writes the slice to the underlying buffer at the offset or returns false if out of range.
	Write(offset uint32, v []byte) bool
}
