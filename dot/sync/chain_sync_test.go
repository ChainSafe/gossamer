// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p-core/peer"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"
)

func Test_chainSyncState_String(t *testing.T) {
	tests := []struct {
		name string
		s    chainSyncState
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_chainSync_determineSyncPeers(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		req        *network.BlockRequestMessage
		peersTried map[peer.ID]struct{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []peer.ID
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if got := cs.determineSyncPeers(tt.args.req, tt.args.peersTried); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("determineSyncPeers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_chainSync_dispatchWorker(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		w *worker
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			fmt.Printf("cs %v\n", cs)
		})
	}
}

func Test_chainSync_doSync(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		req        *network.BlockRequestMessage
		peersTried map[peer.ID]struct{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *workerError
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if got := cs.doSync(tt.args.req, tt.args.peersTried); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("doSync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_chainSync_getTarget(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   *big.Int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if got := cs.getTarget(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_chainSync_handleWork(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		ps *peerState
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if err := cs.handleWork(tt.args.ps); (err != nil) != tt.wantErr {
				t.Errorf("handleWork() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_chainSync_ignorePeer(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		who peer.ID
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			fmt.Printf("cs %v\n", cs)
		})
	}
}

func Test_chainSync_logSyncSpeed(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			fmt.Printf("cs %v\n", cs)
		})
	}
}

func Test_chainSync_maybeSwitchMode(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			fmt.Printf("cs %v\n", cs)
		})
	}
}

func Test_chainSync_setBlockAnnounce(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		from   peer.ID
		header *types.Header
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if err := cs.setBlockAnnounce(tt.args.from, tt.args.header); (err != nil) != tt.wantErr {
				t.Errorf("setBlockAnnounce() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_chainSync_setMode(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		mode chainSyncState
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			fmt.Printf("cs %v\n", cs)
		})
	}
}

func Test_chainSync_setPeerHead(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		p      peer.ID
		hash   common.Hash
		number *big.Int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if err := cs.setPeerHead(tt.args.p, tt.args.hash, tt.args.number); (err != nil) != tt.wantErr {
				t.Errorf("setPeerHead() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_chainSync_start(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			fmt.Printf("cs %v\n", cs)
		})
	}
}

func Test_chainSync_stop(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			fmt.Printf("cs %v\n", cs)
		})
	}
}

func Test_chainSync_sync(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			fmt.Printf("cs %v\n", cs)
		})
	}
}

func Test_chainSync_syncState(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   chainSyncState
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if got := cs.syncState(); got != tt.want {
				t.Errorf("syncState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_chainSync_tryDispatchWorker(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		w *worker
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			fmt.Printf("cs %v\n", cs)
		})
	}
}

func Test_chainSync_validateBlockData(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		req *network.BlockRequestMessage
		bd  *types.BlockData
		p   peer.ID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if err := cs.validateBlockData(tt.args.req, tt.args.bd, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("validateBlockData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_chainSync_validateJustification(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		bd *types.BlockData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if err := cs.validateJustification(tt.args.bd); (err != nil) != tt.wantErr {
				t.Errorf("validateJustification() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_chainSync_validateResponse(t *testing.T) {
	type fields struct {
		ctx              context.Context
		cancel           context.CancelFunc
		blockState       BlockState
		network          Network
		workQueue        chan *peerState
		resultQueue      chan *worker
		RWMutex          sync.RWMutex
		peerState        map[peer.ID]*peerState
		ignorePeers      map[peer.ID]struct{}
		workerState      *workerState
		readyBlocks      *blockQueue
		pendingBlocks    DisjointBlockSet
		state            chainSyncState
		handler          workHandler
		benchmarker      *syncBenchmarker
		finalisedCh      <-chan *types.FinalisationInfo
		minPeers         int
		maxWorkerRetries uint16
		slotDuration     time.Duration
	}
	type args struct {
		req  *network.BlockRequestMessage
		resp *network.BlockResponseMessage
		p    peer.ID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &chainSync{
				ctx:              tt.fields.ctx,
				cancel:           tt.fields.cancel,
				blockState:       tt.fields.blockState,
				network:          tt.fields.network,
				workQueue:        tt.fields.workQueue,
				resultQueue:      tt.fields.resultQueue,
				RWMutex:          tt.fields.RWMutex,
				peerState:        tt.fields.peerState,
				ignorePeers:      tt.fields.ignorePeers,
				workerState:      tt.fields.workerState,
				readyBlocks:      tt.fields.readyBlocks,
				pendingBlocks:    tt.fields.pendingBlocks,
				state:            tt.fields.state,
				handler:          tt.fields.handler,
				benchmarker:      tt.fields.benchmarker,
				finalisedCh:      tt.fields.finalisedCh,
				minPeers:         tt.fields.minPeers,
				maxWorkerRetries: tt.fields.maxWorkerRetries,
				slotDuration:     tt.fields.slotDuration,
			}
			if err := cs.validateResponse(tt.args.req, tt.args.resp, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("validateResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleReadyBlock(t *testing.T) {
	type args struct {
		bd            *types.BlockData
		pendingBlocks DisjointBlockSet
		readyBlocks   *blockQueue
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func Test_newChainSync(t *testing.T) {
	type args struct {
		cfg *chainSyncConfig
	}
	tests := []struct {
		name string
		args args
		want *chainSync
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newChainSync(tt.args.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newChainSync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_workerToRequests(t *testing.T) {
	type args struct {
		w *worker
	}
	tests := []struct {
		name    string
		args    args
		want    []*network.BlockRequestMessage
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := workerToRequests(tt.args.w)
			if (err != nil) != tt.wantErr {
				t.Errorf("workerToRequests() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("workerToRequests() got = %v, want %v", got, tt.want)
			}
		})
	}
}