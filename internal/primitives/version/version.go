// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package version

import (
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

type ApiID [8]uint8

type ApiIDVersion struct {
	ApiId   ApiID
	Version uint32
}
type ApisVec []ApiIDVersion

// Runtime version.
// This should not be thought of as classic Semver (major/minor/tiny).
// This triplet have different semantics and mis-interpretation could cause problems.
// In particular: bug fixes should result in an increment of spec_version and possibly
// authoring_version, absolutely not impl_version since they change the semantics of the
// runtime.
type RuntimeVersion struct {
	// Identifies the different Substrate runtimes. There'll be at least polkadot and node.
	// A different on-chain SpecName to that of the native runtime would normally result
	// in node not attempting to sync or author blocks.
	SpecName string

	// Name of the implementation of the spec. This is of little consequence for the node
	// and serves only to differentiate code of different implementation teams. For this
	// codebase, it will be parity-polkadot. If there were a non-Rust implementation of the
	// Polkadot runtime (e.g. C++), then it would identify itself with an accordingly different
	// impl_name.
	ImplName string

	// AuthoringVersion is the version of the authorship interface. An authoring node
	// will not attempt to author blocks unless this is equal to its native runtime.
	AuthoringVersion uint32

	// Version of the runtime specification.
	//
	// A full-node will not attempt to use its native runtime in substitute for the on-chain
	// Wasm runtime unless all of SpecName, SpecVersion and AuthoringVersion are the same
	// between Wasm and native.
	//
	// This number should never decrease.
	SpecVersion uint32

	// Version of the implementation of the specification.
	//
	// Nodes are free to ignore this; it serves only as an indication that the code is different;
	// as long as the other two versions are the same then while the actual code may be different,
	// it is nonetheless required to do the same thing. Non-consensus-breaking optimizations are
	// about the only changes that could be made which would result in only the ImplVersion
	// changing.
	//
	// This number can be reverted to 0 after a SpecVersion bump.
	ImplVersion uint32

	// List of supported API "features" along with their versions.
	Apis ApisVec

	// All existing calls (dispatchables) are fully compatible when this number doesn't change. If
	// this number changes, then SpecVersion must change, also.
	//
	// This number should never decrease.
	TransactionVersion uint32

	// Version of the system implementation used by this runtime.
	// Use of an incorrect version is consensus breaking.
	SystemVersion uint8
}

func (v *RuntimeVersion) StateVersion() storage.StateVersion {
	// If version > than 1, keep using latest version.
	stateVersion, err := trie.ParseVersion(v.SystemVersion)
	if err != nil {
		return storage.StateVersionV1
	}

	return storage.StateVersion(stateVersion)
}
