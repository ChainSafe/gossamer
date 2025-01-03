package client

import (
	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/dolthub/maphash"
	"github.com/elastic/go-freelru"
)

const notificationPinningLimit = uint32(1024)

type unpinBlock[H runtime.Hash] interface {
	// Pin the block to keep body, justification and state available after pruning.
	// Number of pins are reference counted. Users need to make sure to perform
	// one call to UnpinBlock per call to PinBlock.
	PinBlock(hash H) error

	// Unpin the block to allow pruning.
	UnpinBlock(hash H)
}

type notificationPinningWorker[
	H runtime.Hash,
] struct {
	unpinMessageChan <-chan api.UnpinWorkerMessage[H]
	backend          unpinBlock[H]
	pinnedBlocks     freelru.Cache[H, uint32]
}

func (npw *notificationPinningWorker[H]) handleAnnounceMessage(hash H) {
	count, _ := npw.pinnedBlocks.Peek(hash)
	count++ // this will set to 1 anyways if doesn't exist
	npw.pinnedBlocks.Add(hash, count)
}

func (npw *notificationPinningWorker[H]) handleUnpinMessage(hash H) {
	count, ok := npw.pinnedBlocks.Peek(hash)
	if ok {
		count = count - 1
		npw.pinnedBlocks.Add(hash, count)
		if count == 0 {
			npw.pinnedBlocks.Remove(hash)
		}
		logger.Debugf("Reducing pinning refcount for block hash = %s", hash)
		npw.backend.UnpinBlock(hash)
	} else {
		logger.Debugf("Received unpin message for already unpinned block. hash = %s", hash)
	}
}

func (npw *notificationPinningWorker[H]) run() {
	for msg := range npw.unpinMessageChan {
		switch msg := msg.(type) {
		case api.Unpin[H]:
			npw.handleUnpinMessage(msg.Hash)
		case api.AnnouncePin[H]:
			npw.handleAnnounceMessage(msg.Hash)
		}
	}
	logger.Debugf("Terminating unpin-worker, stream terminated.")
}

type hasher[K comparable] struct {
	maphash.Hasher[K]
}

func (h hasher[K]) Hash(key K) uint32 {
	return uint32(h.Hasher.Hash(key))
}

func newNotificationPinningWorker[H runtime.Hash](
	unpinMesageChan <-chan api.UnpinWorkerMessage[H],
	backend unpinBlock[H],
) *notificationPinningWorker[H] {
	return newNotificationPinningWorkerWithLimit(unpinMesageChan, backend, notificationPinningLimit)
}

func newNotificationPinningWorkerWithLimit[H runtime.Hash](
	unpinMesageChan <-chan api.UnpinWorkerMessage[H],
	backend unpinBlock[H],
	limit uint32,
) *notificationPinningWorker[H] {
	h := hasher[H]{maphash.NewHasher[H]()}
	pinnedBlocks, err := freelru.New[H, uint32](limit, h.Hash)
	if err != nil {
		panic(err)
	}
	pinnedBlocks.SetOnEvict(func(h H, references uint32) {
		if references > 0 {
			logger.Warnf("Notification block pinning limit reached. Unpinning block with hash = %s", h)
			for i := uint32(0); i < references; i++ {
				backend.UnpinBlock(h)
			}
		} else {
			logger.Tracef("Unpinned block. hash = %s", h)
		}
	})
	worker := notificationPinningWorker[H]{
		unpinMessageChan: unpinMesageChan,
		backend:          backend,
		pinnedBlocks:     pinnedBlocks,
	}
	go worker.run()
	return &worker
}
