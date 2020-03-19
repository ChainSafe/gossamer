package core

import (
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/trie"
)

func newTestSyncer(t *testing.T, cfg *SyncerConfig) *Syncer {
	if cfg == nil {
		cfg = &SyncerConfig{}
	}

	cfg.Lock = &sync.Mutex{}

	stateSrvc := state.NewService("")
	stateSrvc.UseMemDB()

	err := stateSrvc.Initialize(genesisHeader, trie.NewEmptyTrie(nil))
	if err != nil {
		t.Fatal(err)
	}

	err = stateSrvc.Start()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.BlockNumberIn == nil {
		cfg.BlockNumberIn = make(chan *big.Int)
	}

	if cfg.MsgIn == nil {
		cfg.MsgIn = make(chan *network.BlockResponseMessage)
	}

	if cfg.MsgOut == nil {
		cfg.MsgOut = make(chan network.Message)
	}

	syncer, err := NewSyncer(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return syncer
}

func TestWatchForBlocks(t *testing.T) {
	blockNumberIn := make(chan *big.Int)
	msgOut := make(chan network.Message)

	cfg := &SyncerConfig{
		BlockNumberIn: blockNumberIn,
		MsgOut:        msgOut,
	}

	syncer := newTestSyncer(t, cfg)
	syncer.Start()

	number := big.NewInt(12)
	blockNumberIn <- number

	var msg network.Message

	select {
	case msg = <-msgOut:
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}

	req, ok := msg.(*network.BlockRequestMessage)
	if !ok {
		t.Fatal("did not get BlockRequestMessage")
	}

	if req.StartingBlock.Value().(uint64) != 1 {
		t.Fatalf("Fail: got %d expected %d", req.StartingBlock.Value(), 1)
	}

	if syncer.requestStart != 1 {
		t.Fatalf("Fail: got %d expected %d", syncer.requestStart, 1)
	}

	if syncer.highestSeenBlock.Cmp(number) != 0 {
		t.Fatalf("Fail: highestSeenBlock=%d expected %d", syncer.highestSeenBlock, number)
	}
}

func TestWatchForBlocks_NotHighestSeen(t *testing.T) {
	blockNumberIn := make(chan *big.Int)

	cfg := &SyncerConfig{
		BlockNumberIn: blockNumberIn,
	}

	syncer := newTestSyncer(t, cfg)
	syncer.Start()

	number := big.NewInt(12)
	blockNumberIn <- number

	if syncer.highestSeenBlock.Cmp(number) != 0 {
		t.Fatalf("Fail: highestSeenBlock=%d expected %d", syncer.highestSeenBlock, number)
	}

	blockNumberIn <- big.NewInt(11)

	if syncer.highestSeenBlock.Cmp(number) != 0 {
		t.Fatalf("Fail: highestSeenBlock=%d expected %d", syncer.highestSeenBlock, number)
	}
}

func TestWatchForBlocks_GreaterThanHighestSeen(t *testing.T) {
	blockNumberIn := make(chan *big.Int)
	msgOut := make(chan network.Message)

	cfg := &SyncerConfig{
		BlockNumberIn: blockNumberIn,
		MsgOut:        msgOut,
	}

	syncer := newTestSyncer(t, cfg)
	syncer.Start()

	number := big.NewInt(12)
	blockNumberIn <- number

	if syncer.highestSeenBlock.Cmp(number) != 0 {
		t.Fatalf("Fail: highestSeenBlock=%d expected %d", syncer.highestSeenBlock, number)
	}

	var msg network.Message

	select {
	case msg = <-msgOut:
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}

	number = big.NewInt(16)
	blockNumberIn <- number

	select {
	case msg = <-msgOut:
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}

	if syncer.highestSeenBlock.Cmp(number) != 0 {
		t.Fatalf("Fail: highestSeenBlock=%d expected %d", syncer.highestSeenBlock, number)
	}

	req, ok := msg.(*network.BlockRequestMessage)
	if !ok {
		t.Fatal("did not get BlockRequestMessage")
	}

	if req.StartingBlock.Value().(uint64) != 12 {
		t.Fatalf("Fail: got %d expected %d", req.StartingBlock.Value(), 12)
	}
}

func TestWatchForResponses(t *testing.T) {
	blockNumberIn := make(chan *big.Int)
	msgIn := make(chan *network.BlockResponseMessage)

	cfg := &SyncerConfig{
		BlockNumberIn: blockNumberIn,
		MsgIn:        msgIn,
	}

	syncer := newTestSyncer(t, cfg)
	syncer.Start()

	addTestBlocksToState(t, 16, syncer.blockState)


}