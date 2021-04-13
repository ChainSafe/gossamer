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
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
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
	Digest         types.Digest
	BestBlock      bool
}

// SubProtocol returns the block-announces sub-protocol
func (bm *BlockAnnounceMessage) SubProtocol() string {
	return blockAnnounceID
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

// Decode the message into a BlockAnnounceMessage
func (bm *BlockAnnounceMessage) Decode(in []byte) error {
	r := &bytes.Buffer{}
	_, _ = r.Write(in)
	h, err := types.NewEmptyHeader().Decode(r)
	if err != nil {
		return err
	}

	bm.ParentHash = h.ParentHash
	bm.Number = h.Number
	bm.StateRoot = h.StateRoot
	bm.ExtrinsicsRoot = h.ExtrinsicsRoot
	bm.Digest = h.Digest
	bestBlock, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	bm.BestBlock = bestBlock == 1
	return nil
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

func decodeBlockAnnounceHandshake(in []byte) (Handshake, error) {
	hs, err := scale.Decode(in, new(BlockAnnounceHandshake))
	if err != nil {
		return nil, err
	}

	return hs.(*BlockAnnounceHandshake), err
}

func decodeBlockAnnounceMessage(in []byte) (NotificationsMessage, error) {
	msg := new(BlockAnnounceMessage)
	err := msg.Decode(in)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// BlockAnnounceHandshake is exchanged by nodes that are beginning the BlockAnnounce protocol
type BlockAnnounceHandshake struct {
	Roles           byte
	BestBlockNumber uint32
	BestBlockHash   common.Hash
	GenesisHash     common.Hash
}

// SubProtocol returns the block-announces sub-protocol
func (hs *BlockAnnounceHandshake) SubProtocol() string {
	return blockAnnounceID
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
func (hs *BlockAnnounceHandshake) Decode(in []byte) error {
	msg, err := scale.Decode(in, hs)
	if err != nil {
		return err
	}

	hs.Roles = msg.(*BlockAnnounceHandshake).Roles
	hs.BestBlockNumber = msg.(*BlockAnnounceHandshake).BestBlockNumber
	hs.BestBlockHash = msg.(*BlockAnnounceHandshake).BestBlockHash
	hs.GenesisHash = msg.(*BlockAnnounceHandshake).GenesisHash
	return nil
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

	if bhs.GenesisHash != s.blockState.GenesisHash() {
		return errors.New("genesis hash mismatch")
	}

	np, ok := s.notificationsProtocols[BlockAnnounceMsgType]
	if !ok {
		// this should never happen.
		return nil
	}

	// don't need to lock here, since function is always called inside the func returned by
	// `createNotificationsMessageHandler` which locks the map beforehand.
	data, ok := np.getHandshakeData(peer)
	if !ok {
		np.handshakeData.Store(peer, &handshakeData{
			received:  true,
			validated: true,
		})
		data, _ = np.getHandshakeData(peer)
	}

	data.handshake = hs

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

	go func() {
		s.syncQueue.handleBlockAnnounceHandshake(bhs.BestBlockNumber, peer)
	}()

	return nil
}

// handleBlockAnnounceMessage handles BlockAnnounce messages
// if some more blocks are required to sync the announced block, the node will open a sync stream
// with its peer and send a BlockRequest message
func (s *Service) handleBlockAnnounceMessage(peer peer.ID, msg NotificationsMessage) error {
	if an, ok := msg.(*BlockAnnounceMessage); ok {
		s.syncQueue.handleBlockAnnounce(an, peer)
		err := s.syncer.HandleBlockAnnounce(an)
		if err != nil {
			return err
		}
	}

	return nil
}
