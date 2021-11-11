// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

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
	Digest         scale.VaryingDataTypeSlice
	BestBlock      bool
}

// SubProtocol returns the block-announces sub-protocol
func (*BlockAnnounceMessage) SubProtocol() string {
	return blockAnnounceID
}

// Type returns BlockAnnounceMsgType
func (*BlockAnnounceMessage) Type() byte {
	return BlockAnnounceMsgType
}

// string formats a BlockAnnounceMessage as a string
func (bm *BlockAnnounceMessage) String() string {
	return fmt.Sprintf("BlockAnnounceMessage ParentHash=%s Number=%d StateRoot=%s ExtrinsicsRoot=%s Digest=%v",
		bm.ParentHash,
		bm.Number,
		bm.StateRoot,
		bm.ExtrinsicsRoot,
		bm.Digest)
}

// Encode a BlockAnnounce Msg Type containing the BlockAnnounceMessage using scale.Encode
func (bm *BlockAnnounceMessage) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*bm)
	if err != nil {
		return enc, err
	}
	return enc, nil
}

// Decode the message into a BlockAnnounceMessage
func (bm *BlockAnnounceMessage) Decode(in []byte) error {
	err := scale.Unmarshal(in, bm)
	if err != nil {
		return err
	}
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
func (*BlockAnnounceMessage) IsHandshake() bool {
	return false
}

func decodeBlockAnnounceHandshake(in []byte) (Handshake, error) {
	hs := BlockAnnounceHandshake{}
	err := scale.Unmarshal(in, &hs)
	if err != nil {
		return nil, err
	}

	return &hs, err
}

func decodeBlockAnnounceMessage(in []byte) (NotificationsMessage, error) {
	msg := BlockAnnounceMessage{
		Number: big.NewInt(0),
		Digest: types.NewDigest(),
	}
	err := msg.Decode(in)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// BlockAnnounceHandshake is exchanged by nodes that are beginning the BlockAnnounce protocol
type BlockAnnounceHandshake struct {
	Roles           byte
	BestBlockNumber uint32
	BestBlockHash   common.Hash
	GenesisHash     common.Hash
}

// SubProtocol returns the block-announces sub-protocol
func (*BlockAnnounceHandshake) SubProtocol() string {
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
	return scale.Marshal(*hs)
}

// Decode the message into a BlockAnnounceHandshake
func (hs *BlockAnnounceHandshake) Decode(in []byte) error {
	err := scale.Unmarshal(in, hs)
	if err != nil {
		return err
	}
	return nil
}

// Type ...
func (*BlockAnnounceHandshake) Type() byte {
	return 0
}

// Hash ...
func (*BlockAnnounceHandshake) Hash() common.Hash {
	return common.Hash{}
}

// IsHandshake returns true
func (*BlockAnnounceHandshake) IsHandshake() bool {
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

func (s *Service) validateBlockAnnounceHandshake(from peer.ID, hs Handshake) error {
	bhs, ok := hs.(*BlockAnnounceHandshake)
	if !ok {
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
	data, ok := np.getInboundHandshakeData(from)
	if ok {
		data.handshake = hs
		np.inboundHandshakeData.Store(from, data)
	}

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

	return s.syncer.HandleBlockAnnounceHandshake(from, bhs)
}

// handleBlockAnnounceMessage handles BlockAnnounce messages
// if some more blocks are required to sync the announced block, the node will open a sync stream
// with its peer and send a BlockRequest message
func (s *Service) handleBlockAnnounceMessage(from peer.ID, msg NotificationsMessage) (propagate bool, err error) {
	bam, ok := msg.(*BlockAnnounceMessage)
	if !ok {
		return false, errors.New("invalid message")
	}

	if err = s.syncer.HandleBlockAnnounce(from, bam); err != nil {
		return false, err
	}

	return true, nil
}
