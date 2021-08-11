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
	ctx           context.Context
	streamDataMap *sync.Map //map[string]*streamData
}

func newStreamManager(ctx context.Context) *streamManager {
	return &streamManager{
		ctx:           ctx,
		streamDataMap: new(sync.Map),
	}
}

func (sm *streamManager) start() {
	go func() {
		ticker := time.NewTicker(cleanupStreamInterval)
		defer ticker.Stop()

		for {
			select {
			case <-sm.ctx.Done():
				return
			case <-ticker.C:
				sm.cleanupStreams()
			}
		}
	}()
}

func (sm *streamManager) cleanupStreams() {
	sm.streamDataMap.Range(func(id, data interface{}) bool {
		sdata := data.(*streamData)
		lastReceived := sdata.lastReceivedMessage
		stream := sdata.stream

		if time.Since(lastReceived) > cleanupStreamInterval {
			_ = stream.Close()
			sm.streamDataMap.Delete(id)
		}

		return true
	})
}

func (sm *streamManager) logNewStream(stream network.Stream) {
	data := &streamData{
		lastReceivedMessage: time.Now(), // prevents closing just opened streams, in case the cleanup goroutine runs at the same time stream is opened
		stream:              stream,
	}
	sm.streamDataMap.Store(stream.ID(), data)
}

func (sm *streamManager) logMessageReceived(streamID string) {
	data, has := sm.streamDataMap.Load(streamID)
	if !has {
		return
	}

	sdata := data.(*streamData)
	sdata.lastReceivedMessage = time.Now()
	sm.streamDataMap.Store(streamID, sdata)
}
