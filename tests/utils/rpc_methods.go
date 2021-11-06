// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package utils

//nolint
var (
	// CHAIN METHODS
	ChainGetBlock                = "chain_getBlock"
	ChainGetHeader               = "chain_getHeader"
	ChainGetFinalizedHead        = "chain_getFinalizedHead"
	ChainGetFinalizedHeadByRound = "chain_getFinalizedHeadByRound"
	ChainGetBlockHash            = "chain_getBlockHash"

	// AUTHOR METHODS
	AuthorSubmitExtrinsic   = "author_submitExtrinsic"
	AuthorPendingExtrinsics = "author_pendingExtrinsics"

	// STATE METHODS
	StateGetStorage = "state_getStorage"

	// DEV METHODS
	DevControl = "dev_control"

	// GRANDPA
	GrandpaProveFinality = "grandpa_proveFinality"
)
