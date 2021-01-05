// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	_ NotificationsMessage = &BlockAnnounceMessage{}
	_ NotificationsMessage = &BlockAnnounceHandshake{}
)

// BlockAnnounceMessage is a state block header
type BlockAnnounceMessage struct {
	ParentHash     common.Hash
	Number         *big.Int
	StateRoot      common.Hash
	ExtrinsicsRoot common.Hash
	Digest         [][]byte // any additional block info eg. logs, seal
}

// Type returns BlockAnnounceMsgType
func (bm *BlockAnnounceMessage) Type() byte {
	return BlockAnnounceMsgType
}

// string formats a BlockAnnounceMessage as a string
func (bm *BlockAnnounceMessage) String() string {
	return fmt.Sprintf("BlockAnnounceMessage ParentHash=%s Number=%d StateRoot=%sx ExtrinsicsRoot=%s Digest=%v",
		bm.ParentHash,
		bm.Number,
		bm.StateRoot,
		bm.ExtrinsicsRoot,
		bm.Digest)
}

// Encode a BlockAnnounce Msg Type containing the BlockAnnounceMessage using scale.Encode
func (bm *BlockAnnounceMessage) Encode() ([]byte, error) {
	enc, err := scale.Encode(bm)
	if err != nil {
		return enc, err
	}
	return enc, nil
}

// Decode the message into a BlockAnnounceMessage, it assumes the type byte has been removed
func (bm *BlockAnnounceMessage) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(bm)
	return err
}

// Hash returns the hash of the BlockAnnounceMessage
func (bm *BlockAnnounceMessage) Hash() common.Hash {
	// scale encode each extrinsic
	encMsg, _ := bm.Encode()
	hash, _ := common.Blake2bHash(encMsg)
	return hash
}

// IsHandshake returns false
func (bm *BlockAnnounceMessage) IsHandshake() bool {
	return false
}

func decodeBlockAnnounceHandshake(r io.Reader) (Handshake, error) {
	sd := scale.Decoder{Reader: r}
	hs := new(BlockAnnounceHandshake)
	_, err := sd.Decode(hs)
	return hs, err
}

func decodeBlockAnnounceMessage(r io.Reader) (NotificationsMessage, error) {
	sd := scale.Decoder{Reader: r}
	msg := new(BlockAnnounceMessage)
	_, err := sd.Decode(msg)
	return msg, err
}

// BlockAnnounceHandshake is exchanged by nodes that are beginning the BlockAnnounce protocol
type BlockAnnounceHandshake struct {
	Roles           byte
	BestBlockNumber uint32
	BestBlockHash   common.Hash
	GenesisHash     common.Hash
}

// String formats a BlockAnnounceHandshake as a string
func (hs *BlockAnnounceHandshake) String() string {
	return fmt.Sprintf("BlockAnnounceHandshake Roles=%d BestBlockNumber=%d BestBlockHash=%s GenesisHash=%s",
		hs.Roles,
		hs.BestBlockNumber,
		hs.BestBlockHash,
		hs.GenesisHash)
}

// Encode encodes a BlockAnnounceHandshake message using SCALE
func (hs *BlockAnnounceHandshake) Encode() ([]byte, error) {
	return scale.Encode(hs)
}

// Decode the message into a BlockAnnounceHandshake
func (hs *BlockAnnounceHandshake) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(hs)
	return err
}

// Type ...
func (hs *BlockAnnounceHandshake) Type() byte {
	return 0
}

// Hash ...
func (hs *BlockAnnounceHandshake) Hash() common.Hash {
	return common.Hash{}
}

// IsHandshake returns true
func (hs *BlockAnnounceHandshake) IsHandshake() bool {
	return true
}

func (s *Service) getBlockAnnounceHandshake() (Handshake, error) {
	latestBlock, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	return &BlockAnnounceHandshake{
		Roles:           s.cfg.Roles,
		BestBlockNumber: uint32(latestBlock.Number.Uint64()),
		BestBlockHash:   latestBlock.Hash(),
		GenesisHash:     s.blockState.GenesisHash(),
	}, nil
}

func (s *Service) validateBlockAnnounceHandshake(peer peer.ID, hs Handshake) error {
	var (
		bhs *BlockAnnounceHandshake
		ok  bool
	)

	if bhs, ok = hs.(*BlockAnnounceHandshake); !ok {
		return errors.New("invalid handshake type")
	}

	// if bhs.GenesisHash != s.blockState.GenesisHash() {
	// 	return errors.New("genesis hash mismatch")
	// }

	// if peer has higher best block than us, begin syncing
	latestHeader, err := s.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	bestBlockNum := big.NewInt(int64(bhs.BestBlockNumber))

	// check if peer block number is greater than host block number
	if latestHeader.Number.Cmp(bestBlockNum) >= 0 {
		return nil
	}

	// if so, send block request
	logger.Info("sending peer highest block to syncer", "number", bhs.BestBlockNumber)
	req := s.syncer.HandleBlockAnnounceHandshake(bestBlockNum)
	if req == nil {
		return nil
	}

	logger.Info("sending block request msg", "number", bhs.BestBlockNumber)
	return s.host.send(peer, syncID, req)
}

// handleBlockAnnounceMessage handles BlockAnnounce messages
// if some more blocks are required to sync the announced block, the node will open a sync stream
// with its peer and send a BlockRequest message
func (s *Service) handleBlockAnnounceMessage(peer peer.ID, msg NotificationsMessage) error {
	if an, ok := msg.(*BlockAnnounceMessage); ok {
		req := s.syncer.HandleBlockAnnounce(an)
		if req != nil {
			s.syncing[peer] = struct{}{}
			err := s.host.send(peer, syncID, req)
			if err != nil {
				logger.Error("failed to send BlockRequest message", "peer", peer)
			}
		}
	}

	return nil
}
