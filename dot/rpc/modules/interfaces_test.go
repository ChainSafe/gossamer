// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

type Runtime interface {
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
