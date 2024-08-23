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
