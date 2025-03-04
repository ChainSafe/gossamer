// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"time"

	"github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot/state"
)

type ServiceConfig func(svc *SyncService)

func WithEpochState(es EpochState) ServiceConfig {
	return func(svc *SyncService) {
		svc.epochState = es
	}
}

func WithGrandpaState(gs GrandpaState) ServiceConfig {
	return func(svc *SyncService) {
		svc.grandpaState = gs
	}
}

func WithStorageState(ss StorageState) ServiceConfig {
	return func(svc *SyncService) {
		svc.storageState = ss
	}
}

func WithFinalityGadget(fg FinalityGadget) ServiceConfig {
	return func(svc *SyncService) {
		svc.finalityGadget = fg
	}
}

func WithBabeVerifier(bv BabeVerifier) ServiceConfig {
	return func(svc *SyncService) {
		svc.babeVerifier = bv
	}
}

func WithBlockImportHandler(bih BlockImportHandler) ServiceConfig {
	return func(svc *SyncService) {
		svc.blockImportHandler = bih
	}
}

func WithTelemetry(t Telemetry) ServiceConfig {
	return func(svc *SyncService) {
		svc.telemetry = t
	}
}

func WithBadBlocks(badBlocks []string) ServiceConfig {
	return func(svc *SyncService) {
		svc.badBlocks = badBlocks
	}
}

func WithSyncMethod(method config.SyncMode) ServiceConfig {
	return func(svc *SyncService) {
		svc.syncStrategy = method
	}
}

func WithTransactionState(ts TransactionState) ServiceConfig {
	return func(svc *SyncService) {
		svc.transactionState = ts
	}
}

func WithNetwork(net Network) ServiceConfig {
	return func(svc *SyncService) {
		svc.network = net
		svc.workerPool = newSyncWorkerPool(net)
	}
}

func WithBlockState(bs state.BlockState) ServiceConfig {
	return func(svc *SyncService) {
		svc.blockState = bs
	}
}

func WithSlotDuration(slotDuration time.Duration) ServiceConfig {
	return func(svc *SyncService) {
		svc.slotDuration = slotDuration
	}
}

func WithMinPeers(min int) ServiceConfig {
	return func(svc *SyncService) {
		svc.minPeers = min
	}
}
