// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

const (
	// v0.9 test API wasm
	HOST_API_TEST_RUNTIME     = "hostapi_runtime"
	HOST_API_TEST_RUNTIME_FP  = "hostapi_runtime.compact.wasm"
	HOST_API_TEST_RUNTIME_URL = "https://raw.githubusercontent.com/kishansagathiya/polkadot-spec/extra-runtimes/test/runtimes/hostapi/hostapi_runtime.compact.wasm?raw=true" //nolint:lll

	// v0.9.29 polkadot
	POLKADOT_RUNTIME_v0929     = "polkadot_runtime-v929"
	POLKADOT_RUNTIME_V0929_FP  = "polkadot_runtime-v929.compact.wasm"
	POLKADOT_RUNTIME_V0929_URL = "https://github.com/paritytech/polkadot/releases/download/v0.9." +
		"29/polkadot_runtime-v9290.compact.compressed.wasm?raw=true"

	// v0.9.29 westend
	WESTEND_RUNTIME_v0929     = "westend_runtime-v929"
	WESTEND_RUNTIME_V0929_FP  = "westend_runtime-v929.compact.wasm"
	WESTEND_RUNTIME_V0929_URL = "https://github.com/paritytech/polkadot/releases/download/v0.9." +
		"29/westend_runtime-v9290.compact.compressed.wasm?raw=true"
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
	// BabeAPIGenerateKeyOwnershipProof is the runtime API call BabeApi_generate_key_ownership_proof
	BabeAPIGenerateKeyOwnershipProof = "BabeApi_generate_key_ownership_proof"
	// BabeAPISubmitReportEquivocationUnsignedExtrinsic is the runtime API call
	// BabeApi_submit_report_equivocation_unsigned_extrinsic
	BabeAPISubmitReportEquivocationUnsignedExtrinsic = "BabeApi_submit_report_equivocation_unsigned_extrinsic"
	// GrandpaSubmitReportEquivocation is the runtime API call GrandpaApi_submit_report_equivocation_unsigned_extrinsic
	GrandpaSubmitReportEquivocation = "GrandpaApi_submit_report_equivocation_unsigned_extrinsic"
	// GrandpaGenerateKeyOwnershipProof is the runtime API call GrandpaApi_generate_key_ownership_proof
	GrandpaGenerateKeyOwnershipProof = "GrandpaApi_generate_key_ownership_proof"
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
	// TransactionPaymentCallAPIQueryCallInfo returns call query call info
	TransactionPaymentCallAPIQueryCallInfo = "TransactionPaymentCallApi_query_call_info"
	// TransactionPaymentCallAPIQueryCallFeeDetails returns call query call fee details
	TransactionPaymentCallAPIQueryCallFeeDetails = "TransactionPaymentCallApi_query_call_fee_details"
)
