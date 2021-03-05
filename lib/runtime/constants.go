// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package runtime

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

//nolint
const (
	// v0.8 substrate runtime
	NODE_RUNTIME     = "node_runtime"
	NODE_RUNTIME_FP  = "node_runtime.compact.wasm"
	NODE_RUNTIME_URL = "https://github.com/noot/substrate/blob/noot/v0.8/target/wasm32-unknown-unknown/release/wbuild/node-runtime/node_runtime.compact.wasm?raw=true"

	// v0.8 polkadot runtime
	POLKADOT_RUNTIME     = "polkadot_runtime"
	POLKADOT_RUNTIME_FP  = "polkadot_runtime.compact.wasm"
	POLKADOT_RUNTIME_URL = "https://github.com/noot/polkadot/blob/noot/v0.8.25/polkadot_runtime.wasm?raw=true"

	// v0.8 test API wasm
	HOST_API_TEST_RUNTIME     = "hostapi_runtime"
	HOST_API_TEST_RUNTIME_FP  = "hostapi_runtime.compact.wasm"
	HOST_API_TEST_RUNTIME_URL = "https://github.com/noot/polkadot-spec/blob/master/test/hostapi_runtime.compact.wasm?raw=true"
)

var (
	// CoreVersion is the runtime API call Core_version
	CoreVersion = "Core_version"
	// CoreInitializeBlock is the runtime API call Core_initialize_block
	CoreInitializeBlock = "Core_initialize_block"
	// CoreExecuteBlock is the runtime API call Core_execute_block
	CoreExecuteBlock = "Core_execute_block"
	// Metadata is the runtime API call Metadata_metadata
	Metadata = "Metadata_metadata"
	// TaggedTransactionQueueValidateTransaction is the runtime API call TaggedTransactionQueue_validate_transaction
	TaggedTransactionQueueValidateTransaction = "TaggedTransactionQueue_validate_transaction"
	// GrandpaAuthorities is the runtime API call GrandpaApi_grandpa_authorities
	GrandpaAuthorities = "GrandpaApi_grandpa_authorities"
	// BabeAPIConfiguration is the runtime API call BabeApi_configuration
	BabeAPIConfiguration = "BabeApi_configuration"
	// BlockBuilderInherentExtrinsics is the runtime API call BlockBuilder_inherent_extrinsics
	BlockBuilderInherentExtrinsics = "BlockBuilder_inherent_extrinsics"
	// BlockBuilderApplyExtrinsic is the runtime API call BlockBuilder_apply_extrinsic
	BlockBuilderApplyExtrinsic = "BlockBuilder_apply_extrinsic"
	// BlockBuilderFinalizeBlock is the runtime API call BlockBuilder_finalize_block
	BlockBuilderFinalizeBlock = "BlockBuilder_finalize_block"
)

// GrandpaAuthoritiesKey is the location of GRANDPA authority data in the storage trie for LEGACY_NODE_RUNTIME and NODE_RUNTIME
var GrandpaAuthoritiesKey, _ = common.HexToBytes("0x3a6772616e6470615f617574686f726974696573")

// BABEPrefix is the prefix for all BABE related storage values
var BABEPrefix, _ = common.Twox128Hash([]byte("Babe"))

// BABEAuthoritiesKey is the location of the BABE authorities in the storage trie for NODE_RUNTIME
func BABEAuthoritiesKey() []byte {
	key, _ := common.Twox128Hash([]byte("Authorities"))
	return append(BABEPrefix, key...)
}

// BABERandomnessKey is the location of the BABE initial randomness in the storage trie for NODE_RUNTIME
func BABERandomnessKey() []byte {
	key, _ := common.Twox128Hash([]byte("Randomness"))
	return append(BABEPrefix, key...)
}

// SystemAccountPrefix is the prefix for all System Account related storage values
func SystemAccountPrefix() []byte {
	// build prefix
	prefix, _ := common.Twox128Hash([]byte(`System`))
	part2, _ := common.Twox128Hash([]byte(`Account`))
	return append(prefix, part2...)
}
