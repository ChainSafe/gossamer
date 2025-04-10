// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package version

import (
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

type ApiId [8]uint8

type ApisVecEntry struct {
	ApiId   ApiId
	Version uint32
}
type ApisVec []ApisVecEntry

type RuntimeVersion struct {
	// Identifies the different Substrate runtimes. There'll be at least polkadot and node.
	// A different on-chain spec_name to that of the native runtime would normally result
	// in node not attempting to sync or author blocks.
	SpecName string

	// Name of the implementation of the spec. This is of little consequence for the node
	// and serves only to differentiate code of different implementation teams. For this
	// codebase, it will be parity-polkadot. If there were a non-Rust implementation of the
	// Polkadot runtime (e.g. C++), then it would identify itself with an accordingly different
	// `impl_name`.
	ImplName string

	// `authoring_version` is the version of the authorship interface. An authoring node
	// will not attempt to author blocks unless this is equal to its native runtime.
	AuthoringVersion uint32

	// Version of the runtime specification.
	//
	// A full-node will not attempt to use its native runtime in substitute for the on-chain
	// Wasm runtime unless all of `spec_name`, `spec_version` and `authoring_version` are the same
	// between Wasm and native.
	//
	// This number should never decrease.
	SpecVersion uint32

	// Version of the implementation of the specification.
	//
	// Nodes are free to ignore this; it serves only as an indication that the code is different;
	// as long as the other two versions are the same then while the actual code may be different,
	// it is nonetheless required to do the same thing. Non-consensus-breaking optimizations are
	// about the only changes that could be made which would result in only the `impl_version`
	// changing.
	//
	// This number can be reverted to `0` after a [`spec_version`](Self::spec_version) bump.
	ImplVersion uint32

	// List of supported API "features" along with their versions.
	Apis ApisVec

	// All existing calls (dispatchables) are fully compatible when this number doesn't change. If
	// this number changes, then [`spec_version`](Self::spec_version) must change, also.
	//
	// This number must change when an existing call (pallet index, call index) is changed,
	// either through an alteration in its user-level semantics, a parameter
	// added/removed, a parameter type changed, or a call/pallet changing its index. An alteration
	// of the user level semantics is for example when the call was before `transfer` and now is
	// `transfer_all`, the semantics of the call changed completely.
	//
	// Removing a pallet or a call doesn't require a *bump* as long as no pallet or call is put at
	// the same index. Removing doesn't require a bump as the chain will reject a transaction
	// referencing this removed call/pallet while decoding and thus, the user isn't at risk to
	// execute any unknown call. FRAME runtime devs have control over the index of a call/pallet
	// to prevent that an index gets reused.
	//
	// Adding a new pallet or call also doesn't require a *bump* as long as they also don't reuse
	// any previously used index.
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
