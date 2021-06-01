package network

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
)

var cleanupStreamInterval = time.Minute

type streamData struct {
	lastReceivedMessage time.Time
	stream              network.Stream
}

// streamManager tracks inbound streams and runs a cleanup goroutine every `cleanupStreamInterval` to close streams that
// we haven't received any data on for the last time period. this prevents keeping stale streams open and continuously trying to
// read from it, which takes up lots of CPU over time.
type streamManager struct {
	ctx        context.Context
	streamData *sync.Map //map[string]streamData
}

func newStreamManager(ctx context.Context) *streamManager {
	return &streamManager{
		ctx:        ctx,
		streamData: new(sync.Map),
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
	sm.streamData.Range(func(id, data interface{}) bool {
		streamData := data.(streamData)
		lastReceived := streamData.lastReceivedMessage
		stream := streamData.stream

		if time.Since(lastReceived) > cleanupStreamInterval {
			_ = stream.Close()
			sm.streamData.Delete(id)
		}

		return true
	})
}

func (sm *streamManager) logNewStream(stream network.Stream) {
	data := streamData{
		lastReceivedMessage: time.Now(), // prevents closing just opened streams, in case the cleanup goroutine runs at the same time stream is opened
		stream:              stream,
	}
	sm.streamData.Store(stream.ID(), data)
}

func (sm *streamManager) logMessageReceived(streamID string) {
	data, has := sm.streamData.Load(streamID)
	if !has {
		return
	}

	streamData := data.(streamData)
	streamData.lastReceivedMessage = time.Now()
	sm.streamData.Store(streamID, streamData)
}
