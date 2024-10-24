// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// Instance for runtime methods
type Instance interface {
	Stop()
	NodeStorage() NodeStorage
	NetworkService() BasicNetwork
	Keystore() *keystore.GlobalKeystore
	Validator() bool
	Exec(function string, data []byte) ([]byte, error)
	SetContextStorage(s Storage)
	GetCodeHash() common.Hash
	Version() (Version, error)
	Metadata() (metadata []byte, err error)
	BabeConfiguration() (*types.BabeConfiguration, error)
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
	BabeGenerateKeyOwnershipProof(slot uint64, authorityID [32]byte) (
		types.OpaqueKeyOwnershipProof, error)
	BabeSubmitReportEquivocationUnsignedExtrinsic(
		equivocationProof types.BabeEquivocationProof,
		keyOwnershipProof types.OpaqueKeyOwnershipProof,
	) error
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
	GrandpaGenerateKeyOwnershipProof(authSetID uint64, authorityID ed25519.PublicKeyBytes) (
		types.GrandpaOpaqueKeyOwnershipProof, error)
	GrandpaSubmitReportEquivocationUnsignedExtrinsic(
		equivocationProof types.GrandpaEquivocationProof, keyOwnershipProof types.GrandpaOpaqueKeyOwnershipProof,
	) error
	ParachainHostPersistedValidationData(
		parachaidID parachaintypes.ParaID,
		assumption parachaintypes.OccupiedCoreAssumption,
	) (*parachaintypes.PersistedValidationData, error)
	ParachainHostValidationCode(parachaidID parachaintypes.ParaID, assumption parachaintypes.OccupiedCoreAssumption,
	) (*parachaintypes.ValidationCode, error)
	ParachainHostValidationCodeByHash(validationCodeHash common.Hash) (*parachaintypes.ValidationCode, error)
	ParachainHostValidators() ([]parachaintypes.ValidatorID, error)
	ParachainHostValidatorGroups() (*parachaintypes.ValidatorGroups, error)
	ParachainHostAvailabilityCores() ([]parachaintypes.CoreState, error)
	ParachainHostCheckValidationOutputs(
		parachainID parachaintypes.ParaID,
		outputs parachaintypes.CandidateCommitments,
	) (bool, error)
	ParachainHostSessionIndexForChild() (parachaintypes.SessionIndex, error)
	ParachainHostCandidatePendingAvailability(
		parachainID parachaintypes.ParaID,
	) (*parachaintypes.CommittedCandidateReceipt, error)
	ParachainHostCandidateEvents() ([]parachaintypes.CandidateEvent, error)
	ParachainHostSessionInfo(sessionIndex parachaintypes.SessionIndex) (*parachaintypes.SessionInfo, error)
	ParachainHostAsyncBackingParams() (*parachaintypes.AsyncBackingParams, error)
	ParachainHostMinimumBackingVotes() (uint32, error)
	ParachainHostSessionExecutorParams(index parachaintypes.SessionIndex,
	) (*parachaintypes.ExecutorParams, error)
}
