// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import "github.com/ChainSafe/gossamer/internal/primitives/externalities"

/// The context in which a call is done.
///
/// Depending on the context the executor may chooses different kind of heap sizes for the runtime
/// instance.
// #[derive(Clone, Copy, Debug, PartialEq, Eq, Ord, PartialOrd)]
// pub enum CallContext {
// 	/// The call is happening in some offchain context.
// 	Offchain,
// 	/// The call is happening in some on-chain context like building or importing a block.
// 	Onchain,
// }

// / Code execution engine.
// pub trait CodeExecutor: Sized + Send + Sync + ReadRuntimeVersion + Clone + 'static {
type CodeExecutor interface {
	ReadRuntimeVersion

	// Call a given method in the runtime.
	// Returns a tuple of the result (either the output data or an execution error) together with a
	// bool, which is true if native execution was used.
	Call(
		ext externalities.Externalities,
		runtimeCode RuntimeCode,
		method string,
		data []byte,
		context CallContext,
	) (result []byte, native bool, err error)
}

// / A trait that allows reading version information from the binary.
// pub trait ReadRuntimeVersion: Send + Sync {
type ReadRuntimeVersion interface {
	// /// Reads the runtime version information from the given wasm code.
	// ///
	// /// The version information may be embedded into the wasm binary itself. If it is not present,
	// /// then this function may fallback to the legacy way of reading the version.
	// ///
	// /// The legacy mechanism involves instantiating the passed wasm runtime and calling
	// /// `Core_version` on it. This is a very expensive operation.
	// ///
	// /// `ext` is only needed in case the calling into runtime happens. Otherwise it is ignored.
	// ///
	// /// Compressed wasm blobs are supported and will be decompressed if needed. If uncompression
	// /// fails, the error is returned.
	// ///
	// /// # Errors
	// ///
	// /// If the version information present in binary, but is corrupted - returns an error.
	// ///
	// /// Otherwise, if there is no version information present, and calling into the runtime takes
	// /// place, then an error would be returned if `Core_version` is not provided.
	// fn read_runtime_version(
	//
	//	&self,
	//	wasm_code: &[u8],
	//	ext: &mut dyn Externalities,
	//
	// ) -> Result<Vec<u8>, String>;
}
