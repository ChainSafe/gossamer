// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import "time"

type ServiceConfig func(svc *SyncService)

func WithStrategies(currentStrategy, defaultStrategy Strategy) ServiceConfig {
	return func(svc *SyncService) {
		svc.currentStrategy = currentStrategy
		svc.defaultStrategy = defaultStrategy
	}
}

func WithNetwork(net Network) ServiceConfig {
	return func(svc *SyncService) {
		svc.network = net
		svc.workerPool = newSyncWorkerPool(net)
	}
}

func WithBlockState(bs BlockState) ServiceConfig {
	return func(svc *SyncService) {
		svc.blockState = bs
	}
}

func WithSlotDuration(slotDuration time.Duration) ServiceConfig {
	return func(svc *SyncService) {
		svc.slotDuration = slotDuration
	}
}
