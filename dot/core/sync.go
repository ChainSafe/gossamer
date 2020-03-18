package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	mrand "math/rand"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/rand"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	log "github.com/ChainSafe/log15"
)

// Syncer deals with chain syncing by sending block request messages and watching for responses.
type Syncer struct {
	blockState       BlockState                           // retrieve our current head of chain from BlockState
	blockNumberIn    <-chan *big.Int                      // incoming block numbers seen from other nodes that are higher than ours
	msgOut           chan<- network.Message               // channel to send BlockRequest messages to network service
	msgIn            <-chan *network.BlockResponseMessage // channel to receive BlockResponse messages from
	lock             *sync.Mutex
	synced           bool
	requestStart     int64 // block number from which to begin block requests
	highestSeenBlock *big.Int
}

// SyncerConfig is the configuration for the Syncer.
type SyncerConfig struct {
	BlockState    BlockState
	BlockNumberIn <-chan *big.Int
	MsgIn         <-chan *network.BlockResponseMessage
	MsgOut        chan<- network.Message
	Lock          *sync.Mutex
}

// NewSyncer returns a new Syncer
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
		blockState:       cfg.BlockState,
		blockNumberIn:    cfg.BlockNumberIn,
		msgIn:            cfg.MsgIn,
		msgOut:           cfg.MsgOut,
		lock:             cfg.Lock,
		synced:           true,
		highestSeenBlock: big.NewInt(0),
	}, nil
}

// Start begins the syncer
func (s *Syncer) Start() {
	go s.watchForBlocks()
	go s.watchForResponses()
}

func (s *Syncer) watchForBlocks() {
	for {
		blockNum := <-s.blockNumberIn
		if blockNum != nil && s.highestSeenBlock.Cmp(blockNum) == -1 {
			s.highestSeenBlock = blockNum
			s.requestStart = blockNum.Int64()

			if s.synced {
				s.synced = false
				s.lock.Lock()
			}

			go s.sendBlockRequest()
		}
	}
}

func (s *Syncer) watchForResponses() {
	for {
		msg := <-s.msgIn
		log.Info("[sync] got BlockResponseMessage")

		// highestInResp will be the highest block in the response
		// it's set to 0 if err != nil
		highestInResp, err := s.handleBlockResponse(msg)
		if err != nil {

			// if we cannot find the parent block in our blocktree, we are missing some blocks, and need to request
			// blocks from farther back in the chain
			if err.Error() == "cannot find parent block in blocktree" {
				// set request start
				s.requestStart = s.requestStart - maxResponseSize
				if s.requestStart < 0 {
					s.requestStart = 1
				}
				log.Info("[sync] retrying request", "start", s.requestStart)
				go s.sendBlockRequest()
			} else {
				log.Error("[sync]", "error", err)
			}

		} else {
			// TODO: max retries before unlocking, in case no response is received

			bestNum, err := s.blockState.BestBlockNumber()
			if err != nil {
				log.Error("[sync] Failed to get best block number", "error", err)

				if !s.synced {
					s.lock.Unlock()
				}

				return
			}

			// check if we are synced or not
			if bestNum.Cmp(s.highestSeenBlock) == 0 && bestNum.Cmp(big.NewInt(0)) != 0 {
				log.Info("[sync] All synced up!", "number", bestNum)

				if !s.synced {
					s.lock.Unlock()
					s.synced = true
				}

				//return
			} else {
				// not yet synced, send another block request for the following blocks
				s.requestStart = highestInResp + 1
				go s.sendBlockRequest()
			}

		}

	}
}

func (s *Syncer) sendBlockRequest() {
	//generate random ID
	s1 := rand.NewSource(uint64(time.Now().UnixNano()))
	seed := rand.New(s1).Uint64()
	randomID := mrand.New(mrand.NewSource(int64(seed))).Uint64()

	buf := make([]byte, 8)
	start := uint64(s.requestStart)
	binary.LittleEndian.PutUint64(buf, start)

	log.Info("[sync] block request start", "num", start)

	blockRequest := &network.BlockRequestMessage{
		ID:            randomID, // random
		RequestedData: 3,        // block header + body
		StartingBlock: variadic.NewUint64OrHash(append([]byte{1}, buf...)),
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     1,
		Max:           optional.NewUint32(false, 0),
	}

	// send block request message to network service
	s.msgOut <- blockRequest
}

func (s *Syncer) handleBlockResponse(msg *network.BlockResponseMessage) (int64, error) {
	blockData := msg.BlockData
	highestInResp := int64(0)

	for _, bd := range blockData {
		if bd.Header.Exists() {
			header, err := types.NewHeaderFromOptional(bd.Header)
			if err != nil {
				return 0, err
			}

			// get block header; if exists, return
			existingHeader, err := s.blockState.GetHeader(bd.Hash)
			if err != nil && existingHeader == nil {
				err = s.blockState.SetHeader(header)
				if err != nil {
					return 0, err
				}

				log.Info("[sync] saved block header", "hash", header.Hash(), "number", header.Number)

				// TODO: handle consensus digest, if first in epoch
			}

			if header.Number.Int64() > highestInResp {
				highestInResp = header.Number.Int64()
			}
		}

		if bd.Header.Exists() && bd.Body.Exists {
			header, err := types.NewHeaderFromOptional(bd.Header)
			if err != nil {
				return 0, err
			}

			body, err := types.NewBodyFromOptional(bd.Body)
			if err != nil {
				return 0, err
			}

			block := &types.Block{
				Header: header,
				Body:   body,
			}

			// TODO: execute block and verify authorship right

			err = s.blockState.AddBlock(block)
			if err != nil {
				if err.Error() == "cannot find parent block in blocktree" {
					return 0, err
				} else if strings.Contains(err.Error(), "cannot add block to blocktree that already exists") {
					// this is fine
				} else {
					return 0, err
				}
			} else {
				log.Info("[sync] imported block", "number", header.Number, "hash", header.Hash())
			}
		}

		err := s.compareAndSetBlockData(bd)
		if err != nil {
			return highestInResp, err
		}
	}

	return highestInResp, nil
}

func (s *Syncer) compareAndSetBlockData(bd *types.BlockData) error {
	if s.blockState == nil {
		return fmt.Errorf("no blockState")
	}

	existingData, err := s.blockState.GetBlockData(bd.Hash)
	if err != nil {
		// no block data exists, ok
		return s.blockState.SetBlockData(bd)
	}

	if existingData == nil {
		return s.blockState.SetBlockData(bd)
	}

	if existingData.Header == nil || (!existingData.Header.Exists() && bd.Header.Exists()) {
		existingData.Header = bd.Header
	}

	if existingData.Body == nil || (!existingData.Body.Exists && bd.Body.Exists) {
		existingData.Body = bd.Body
	}

	if existingData.Receipt == nil || (!existingData.Receipt.Exists() && bd.Receipt.Exists()) {
		existingData.Receipt = bd.Receipt
	}

	if existingData.MessageQueue == nil || (!existingData.MessageQueue.Exists() && bd.MessageQueue.Exists()) {
		existingData.MessageQueue = bd.MessageQueue
	}

	if existingData.Justification == nil || (!existingData.Justification.Exists() && bd.Justification.Exists()) {
		existingData.Justification = bd.Justification
	}

	return s.blockState.SetBlockData(existingData)
}
