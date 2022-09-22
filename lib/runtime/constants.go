// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

//nolint:revive
const (
	// v0.9 substrate runtime
	NODE_RUNTIME     = "node_runtime"
	NODE_RUNTIME_FP  = "node_runtime.compact.wasm"
	NODE_RUNTIME_URL = "https://github.com/noot/substrate/blob/noot/v0.9/target/debug/wbuild/node-runtime/node_runtime.compact.wasm?raw=true" //nolint:lll

	// v0.9.8 substrate runtime
	NODE_RUNTIME_v098     = "node_runtime-v0.9.8"
	NODE_RUNTIME_FP_v098  = "node_runtime-v0.9.8.compact.wasm"
	NODE_RUNTIME_URL_v098 = "https://github.com/noot/substrate/blob/noot/v0.9.8/target/debug/wbuild/node-runtime/node_runtime.compact.wasm?raw=true" //nolint:lll

	// v0.9.10 polkadot runtime
	POLKADOT_RUNTIME_v0910     = "polkadot_runtime-v9100"
	POLKADOT_RUNTIME_FP_v0910  = "polkadot_runtime-v9100.compact.wasm"
	POLKADOT_RUNTIME_URL_v0910 = "https://github.com/paritytech/polkadot/releases/download/v0.9.10/polkadot_runtime-v9100.compact.wasm?raw=true" //nolint:lll

	// v0.9.17 polkadot runtime
	POLKADOT_RUNTIME_v0917     = "polkadot_runtime-v917"
	POLKADOT_RUNTIME_FP_v0917  = "polkadot_runtime-v917.compact.wasm"
	POLKADOT_RUNTIME_URL_v0917 = "https://github.com/paritytech/polkadot/blob/c583355a599200d64e9202d05f47860d824b4dcf/target/release/wbuild/polkadot-runtime/polkadot_runtime.compact.wasm?raw=true" //nolint:lll

	// v0.9.25 polkadot runtime
	POLKADOT_RUNTIME_v0925     = "polkadot_runtime-v9250"
	POLKADOT_RUNTIME_FP_v0925  = "polkadot_runtime-v9250.compact.compressed.wasm"
	POLKADOT_RUNTIME_URL_v0925 = "https://github.com/paritytech/polkadot/releases/download/v0.9.25/polkadot_runtime-v9250.compact.compressed.wasm?raw=true" //nolint:lll

	// v0.8 polkadot runtime
	POLKADOT_RUNTIME     = "polkadot_runtime"
	POLKADOT_RUNTIME_FP  = "polkadot_runtime.compact.wasm"
	POLKADOT_RUNTIME_URL = "https://github.com/noot/polkadot/blob/noot/v0.8.25/polkadot_runtime.wasm?raw=true"

	// v0.9 test API wasm
	HOST_API_TEST_RUNTIME     = "hostapi_runtime"
	HOST_API_TEST_RUNTIME_FP  = "hostapi_runtime.compact.wasm"
	HOST_API_TEST_RUNTIME_URL = "https://github.com/ChainSafe/polkadot-spec/blob/4d190603d21d4431888bcb1ec546c4dc03b7bf93/test/runtimes/hostapi/hostapi_runtime.compact.wasm?raw=true" //nolint:lll

	// v0.8 substrate runtime with modified name and babe C=(1, 1)
	DEV_RUNTIME     = "dev_runtime"
	DEV_RUNTIME_FP  = "dev_runtime.compact.wasm"
	DEV_RUNTIME_URL = "https://github.com/noot/substrate/blob/noot/v0.8-dev-runtime/target/wasm32-unknown-unknown/release/wbuild/node-runtime/node_runtime.compact.wasm?raw=true" //nolint:lll
)

const (
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
	// DecodeSessionKeys is the runtime API call SessionKeys_decode_session_keys
	DecodeSessionKeys = "SessionKeys_decode_session_keys"
	// TransactionPaymentAPIQueryInfo returns information of a given extrinsic
	TransactionPaymentAPIQueryInfo = "TransactionPaymentApi_query_info"
)
