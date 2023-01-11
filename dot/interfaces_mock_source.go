// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// babeServiceBuilder interface to define the building of babe service
type babeServiceBuilder interface {
	NewServiceIFace(cfg *babe.ServiceConfig) (service *babe.Service, err error)
}

type nodeBuilderIface interface {
	isNodeInitialised(basepath string) error
	initNode(config *Config) error
	createStateService(config *Config) (*state.Service, error)
	createNetworkService(cfg *Config, stateSrvc *state.Service, telemetryMailer telemetryClient) (*network.Service,
		error)
	createRuntimeStorage(st *state.Service) (*runtime.NodeStorage, error)
	loadRuntime(cfg *Config, ns *runtime.NodeStorage, stateSrvc *state.Service, ks *keystore.GlobalKeystore,
		net *network.Service) error
	createBlockVerifier(st *state.Service) *babe.VerificationManager
	createDigestHandler(lvl log.Level, st *state.Service) (*digest.Handler, error)
	createCoreService(cfg *Config, ks *keystore.GlobalKeystore, st *state.Service, net *network.Service,
		dh *digest.Handler) (*core.Service, error)
	createGRANDPAService(cfg *Config, st *state.Service, ks keyStore,
		net *network.Service, telemetryMailer telemetryClient) (*grandpa.Service, error)
	newSyncService(cfg *Config, st *state.Service, finalityGadget blockJustificationVerifier,
		verifier *babe.VerificationManager, cs *core.Service, net *network.Service,
		telemetryMailer telemetryClient) (*sync.Service, error)
	createBABEService(cfg *Config, st *state.Service, ks keyStore, cs *core.Service,
		telemetryMailer telemetryClient) (service *babe.Service, err error)
	createSystemService(cfg *types.SystemInfo, stateSrvc *state.Service) (*system.Service, error)
	createRPCService(params rpcServiceSettings) (*rpc.HTTPServer, error)
}
