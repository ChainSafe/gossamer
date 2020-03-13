package core

import (
	"encoding/binary"
	"errors"
	"math/big"
	mrand "math/rand"
	"sync"
	"time"

	"golang.org/x/exp/rand"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"

	log "github.com/ChainSafe/log15"
)

type Syncer struct {
	blockState    BlockState             // retrieve our current head of chain from BlockState
	blockNumberIn <-chan *big.Int        // incoming block numbers seen from other nodes that are higher than ours
	msgOut        chan<- network.Message // channel to send message to network service
	lock          sync.Mutex
}

type SyncerConfig struct {
	BlockState    BlockState
	BlockNumberIn <-chan *big.Int
	MsgOut        chan<- network.Message
	Lock          sync.Mutex
}

func NewSyncer(cfg *SyncerConfig) (*Syncer, error) {
	if cfg.BlockState == nil {
		return nil, errors.New("cannot have nil BlockState")
	}

	if cfg.BlockNumberIn == nil {
		return nil, errors.New("cannot have nil BlockNumberIn channel")
	}

	if cfg.MsgOut == nil {
		return nil, errors.New("cannot have nil MsgOut channel")
	}

	return &Syncer{
		blockState:    cfg.BlockState,
		blockNumberIn: cfg.BlockNumberIn,
		msgOut:        cfg.MsgOut,
		lock:          cfg.Lock,
	}, nil
}

func (s *Syncer) Start() {
	go s.watchForBlocks()
}

func (s *Syncer) watchForBlocks() {
	for {
		peerNum := <-s.blockNumberIn

		s.lock.Lock()

		err := s.sendBlockRequest()
		if err != nil {
			log.Error("[sync] watch for blocks", "error", err)
		}

		go s.watchForResponses(peerNum)
	}
}

func (s *Syncer) watchForResponses(peerNum *big.Int) {
	for {
		bestNum, err := s.blockState.BestBlockNumber()
		if err != nil {
			log.Error("[sync] watchForResponses", "error", err)

			s.lock.Unlock()
			return
		}

		if bestNum.Cmp(peerNum) == 0 {
			log.Info("[sync] all synced up!", "number", bestNum)

			s.lock.Unlock()
			return
		}

		time.Sleep(time.Second)
	}
}

func (s *Syncer) sendBlockRequest() error {
	bestNum, err := s.blockState.BestBlockNumber()
	if err != nil {
		log.Error("[sync] sendBlockRequest", "error", err)
		return err
	}

	//generate random ID
	s1 := rand.NewSource(uint64(time.Now().UnixNano()))
	seed := rand.New(s1).Uint64()
	randomID := mrand.New(mrand.NewSource(int64(seed))).Uint64()

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(bestNum.Int64()))

	blockRequest := &network.BlockRequestMessage{
		ID: randomID, // random
		// TODO: figure out what we actually want to request
		RequestedData: 3, // block header + body
		StartingBlock: append([]byte{1}, buf...),
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     1,
		Max:           optional.NewUint32(false, 0),
	}

	// send block request message to network service
	log.Debug("send blockRequest message to network service")

	s.msgOut <- blockRequest

	return nil
}
