// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

var errInvalidRole = errors.New("invalid handshake role")
var (
	_ NotificationsMessage = &BlockAnnounceMessage{}
	_ NotificationsMessage = &BlockAnnounceHandshake{}

	errExpectedBlockAnnounceMsg = errors.New("received block announce handshake but expected block announce message")
)

// BlockAnnounceMessage is a state block header
type BlockAnnounceMessage struct {
	ParentHash     common.Hash
	Number         uint
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
func (bm *BlockAnnounceMessage) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := bm.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
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
	Roles           Roles
	BestBlockNumber uint32
	BestBlockHash   common.Hash
	GenesisHash     common.Hash
}

// Roles is the type of node.
type Roles byte

const (
	// FullNode allow you to read the current state of the chain and to submit and validate
	// extrinsics directly on the network without relying on a centralised infrastructure provider.
	FullNode Roles = 1
	// LightClient node has only the runtime and the current state, but does not store past
	// blocks and so cannot read historical data without requesting it from a node that has it.
	LightClient Roles = 2
	// Validator node helps seal new blocks.
	Validator Roles = 4
)

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

// Hash returns blake2b hash of block announce handshake.
func (hs *BlockAnnounceHandshake) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := hs.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode handshake: %w", err)
	}

	return common.Blake2bHash(encMsg)
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
		Roles:           Roles(s.cfg.Roles),
		BestBlockNumber: uint32(latestBlock.Number),
		BestBlockHash:   latestBlock.Hash(),
		GenesisHash:     s.blockState.GenesisHash(),
	}, nil
}

func (s *Service) validateBlockAnnounceHandshake(from peer.ID, hs Handshake) error {
	bhs, ok := hs.(*BlockAnnounceHandshake)
	if !ok {
		return errors.New("invalid handshake type")
	}

	switch bhs.Roles {
	case FullNode, LightClient, Validator:
	default:
		return errInvalidRole
	}

	if !bhs.GenesisHash.Equal(s.blockState.GenesisHash()) {
		s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
			Value:  peerset.GenesisMismatch,
			Reason: peerset.GenesisMismatchReason,
		}, from)
		return errors.New("genesis hash mismatch")
	}

	np, ok := s.notificationsProtocols[BlockAnnounceMsgType]
	if !ok {
		// this should never happen.
		return nil
	}

	// don't need to lock here, since function is always called inside the func returned by
	// `createNotificationsMessageHandler` which locks the map beforehand.
	data := np.peersData.getInboundHandshakeData(from)
	if data != nil {
		data.handshake = hs
		np.peersData.setInboundHandshakeData(from, data)
	}

	// if peer has higher best block than us, begin syncing
	latestHeader, err := s.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	// check if peer block number is greater than host block number
	if latestHeader.Number >= uint(bhs.BestBlockNumber) {
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
