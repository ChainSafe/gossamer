// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// service can be started and stopped.
type service interface {
	Start() error
	Stop() error
}

// serviceRegisterer can register a service interface, start or stop all services,
// and get a particular service.
type serviceRegisterer interface {
	RegisterService(service services.Service)
	StartAll()
	StopAll()
	Get(srvc interface{}) services.Service
}

// BlockJustificationVerifier has a verification method for block justifications.
type BlockJustificationVerifier interface {
	VerifyBlockJustification(common.Hash, []byte) ([]byte, error)
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}

// Note: only used internally in `loadRuntime` and
// `createRuntime`.
type runtimeInterface interface {
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
	Metadata() ([]byte, error)
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
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
}
