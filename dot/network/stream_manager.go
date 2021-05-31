package network

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
)

var cleanupStreamInterval = time.Minute

// streamManager tracks inbound streams and runs a cleanup goroutine every `cleanupStreamInterval` to close streams that
// we haven't received any data on for the last time period. this prevents keeping stale streams open and continuously trying to
// read from it, which takes up lots of CPU over time.
type streamManager struct {
	ctx                 context.Context
	lastReceivedMessage *sync.Map //map[string]time.Time
	streams             *sync.Map //map[string]network.Stream
}

func newStreamManager(ctx context.Context) *streamManager {
	return &streamManager{
		ctx:                 ctx,
		lastReceivedMessage: new(sync.Map),
		streams:             new(sync.Map),
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
	sm.streams.Range(func(id, stream interface{}) bool {
		lastReceived, has := sm.lastReceivedMessage.Load(id)
		if !has {
			_ = stream.(network.Stream).Close()
			sm.streams.Delete(id)
		}

		if time.Since(lastReceived.(time.Time)) > cleanupStreamInterval {
			_ = stream.(network.Stream).Close()
			sm.streams.Delete(id)
			sm.lastReceivedMessage.Delete(id)
		}

		return true
	})
}

func (sm *streamManager) logNewStream(stream network.Stream) {
	sm.lastReceivedMessage.Store(stream.ID(), time.Now()) // prevents closing just opened streams, in case the cleanup goroutine runs at the same time stream is opened
	sm.streams.Store(stream.ID(), stream)
}

func (sm *streamManager) logMessageReceived(streamID string) {
	sm.lastReceivedMessage.Store(streamID, time.Now())
}
