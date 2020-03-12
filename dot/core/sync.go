package core

import (
	"encoding/binary"
	"errors"
	"math/big"
	mrand "math/rand"
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
}

type SyncerConfig struct {
	BlockState    BlockState
	BlockNumberIn <-chan *big.Int
	MsgOut        chan<- network.Message
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
	}, nil
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

	// //track request
	// s.requestedBlockIDs[randomID] = true

	// send block request message to network service
	log.Debug("send blockRequest message to network service")

	s.msgOut <- blockRequest

	// err = s.safeMsgSend(blockRequest)
	// if err != nil {
	// 	return err
	// }

	return nil
}
