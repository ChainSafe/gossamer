// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// RuntimeInstance for runtime methods
type RuntimeInstance interface {
	UpdateRuntimeCode([]byte) error
	Stop()
	NodeStorage() runtime.NodeStorage
	NetworkService() runtime.BasicNetwork
	Keystore() *keystore.GlobalKeystore
	Validator() bool
	Exec(function string, data []byte) ([]byte, error)
	SetContextStorage(s runtime.Storage)
	GetCodeHash() common.Hash
	Version() (version runtime.Version)
	Metadata() (metadata []byte, err error)
	BabeConfigurer
	GrandpaAuthorities() ([]types.Authority, error)
	ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error)
	InitializeBlock(header *types.Header) error
	InherentExtrinsics(data []byte) ([]byte, error)
	ApplyExtrinsic(data types.Extrinsic) ([]byte, error)
	FinalizeBlock() (*types.Header, error)
	ExecuteBlock(block *types.Block) ([]byte, error)
	DecodeSessionKeys(enc []byte) ([]byte, error)
	PaymentQueryInfo(ext []byte) (*types.RuntimeDispatchInfo, error)
	CheckInherents()
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
	GrandpaGenerateKeyOwnershipProof(authSetID uint64, authorityID ed25519.PublicKeyBytes) (
		types.OpaqueKeyOwnershipProof, error)
	GrandpaSubmitReportEquivocationUnsignedExtrinsic(
		equivocationProof types.GrandpaEquivocationProof, keyOwnershipProof types.OpaqueKeyOwnershipProof,
	) error
}

// BabeConfigurer returns the babe configuration of the runtime.
type BabeConfigurer interface {
	BabeConfiguration() (*types.BabeConfiguration, error)
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}
