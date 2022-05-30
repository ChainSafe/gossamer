// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
)

type streamData struct {
	lastReceivedMessage time.Time
	stream              network.Stream
}

// streamManager tracks inbound streams and runs a cleanup goroutine every `cleanupStreamInterval` to close streams that
// we haven't received any data on for the last time period.
// This prevents keeping stale streams open and continuously trying to
// read from it, which takes up lots of CPU over time.
type streamManager struct {
	ctx                   context.Context
	streamDataMu          sync.Mutex
	streamData            map[string]*streamData
	cleanupStreamInterval time.Duration
}

func newStreamManager(ctx context.Context, cleanupStreamInterval time.Duration) *streamManager {
	return &streamManager{
		ctx:        ctx,
		streamData: make(map[string]*streamData),
	}
}

func (sm *streamManager) start() {
	go func() {
		ticker := time.NewTicker(sm.cleanupStreamInterval)
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
	sm.streamDataMu.Lock()
	defer sm.streamDataMu.Unlock()

	for id, data := range sm.streamData {
		lastReceived := data.lastReceivedMessage
		stream := data.stream

		if time.Since(lastReceived) > sm.cleanupStreamInterval {
			_ = stream.Close()
			delete(sm.streamData, id)
		}
	}
}

func (sm *streamManager) logNewStream(stream network.Stream) {
	data := &streamData{
		// prevents closing just opened streams, in case the cleanup
		// goroutine runs at the same time stream is opened
		lastReceivedMessage: time.Now(),
		stream:              stream,
	}

	sm.streamDataMu.Lock()
	defer sm.streamDataMu.Unlock()
	sm.streamData[stream.ID()] = data
}

func (sm *streamManager) logMessageReceived(streamID string) {
	sm.streamDataMu.Lock()
	defer sm.streamDataMu.Unlock()

	data := sm.streamData[streamID]
	if data == nil {
		return
	}

	data.lastReceivedMessage = time.Now()
}
