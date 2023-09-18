// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	_ NotificationsMessage = &BlockAnnounceMessage{}
	_ Handshake            = (*BlockAnnounceHandshake)(nil)
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

// Type returns blockAnnounceMsgType
func (*BlockAnnounceMessage) Type() MessageType {
	return blockAnnounceMsgType
}

// String formats a BlockAnnounceMessage as a string
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
	Roles           common.NetworkRole
	BestBlockNumber uint32
	BestBlockHash   common.Hash
	GenesisHash     common.Hash
}

// String formats a BlockAnnounceHandshake as a string
func (hs *BlockAnnounceHandshake) String() string {
	return fmt.Sprintf("BlockAnnounceHandshake NetworkRole=%d BestBlockNumber=%d BestBlockHash=%s GenesisHash=%s",
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

// IsValid returns true if handshakes's role is valid.
func (hs *BlockAnnounceHandshake) IsValid() bool {
	switch hs.Roles {
	case common.AuthorityRole, common.FullNodeRole, common.LightClientRole:
		return true
	default:
		return false
	}
}

func (s *Service) getBlockAnnounceHandshake() (Handshake, error) {
	latestBlock, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, err
	}

	return &BlockAnnounceHandshake{
		Roles:           s.cfg.Roles,
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
	case common.FullNodeRole, common.LightClientRole, common.AuthorityRole:
	default:
		return fmt.Errorf("%w: %d", errInvalidRole, bhs.Roles)
	}

	if bhs.GenesisHash != s.blockState.GenesisHash() {
		s.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
			Value:  peerset.GenesisMismatch,
			Reason: peerset.GenesisMismatchReason,
		}, from)
		return errors.New("genesis hash mismatch")
	}

	np, ok := s.notificationsProtocols[blockAnnounceMsgType]
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

	err = s.syncer.HandleBlockAnnounce(from, bam)
	if errors.Is(err, blocktree.ErrBlockExists) {
		return true, nil
	}

	return false, err
}
