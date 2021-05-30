package network

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
)

var cleanupStreamInterval = time.Minute

type streamManager struct {
	ctx                 context.Context
	lastReceivedMessage map[string]time.Time
	streams             map[string]network.Stream
}

func newStreamManager(ctx context.Context) *streamManager {
	return &streamManager{
		ctx:                 ctx,
		lastReceivedMessage: make(map[string]time.Time),
		streams:             make(map[string]network.Stream),
	}
}

func (sm *streamManager) start() {
	go func() {
		for {
			select {
			case <-sm.ctx.Done():
				return
			case <-time.After(cleanupStreamInterval):
				sm.cleanupStreams()
			}
		}
	}()
}

func (sm *streamManager) cleanupStreams() {
	for id, stream := range sm.streams {
		lastReceived, has := sm.lastReceivedMessage[id]
		if !has {
			_ = stream.Close()
			delete(sm.streams, id)
		}

		if time.Since(lastReceived) > cleanupStreamInterval {
			_ = stream.Close()
			delete(sm.streams, id)
			delete(sm.lastReceivedMessage, id)
		}
	}
}

func (sm *streamManager) logNewStream(stream network.Stream) {
	sm.lastReceivedMessage[stream.ID()] = time.Now() // prevents closing just opened streams, in case the cleanup goroutine runs at the same time stream is opened
	sm.streams[stream.ID()] = stream
}

func (sm *streamManager) logMessageReceived(streamID string) {
	sm.lastReceivedMessage[streamID] = time.Now()
}
