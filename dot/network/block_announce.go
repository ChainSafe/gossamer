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

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

func decodeBlockAnnounceHandshake(r io.Reader) (Handshake, error) {
	sd := scale.Decoder{Reader: r}
	hs := new(BlockAnnounceHandshake)
	_, err := sd.Decode(hs)
	return hs, err
}

func decodeBlockAnnounceMessage(r io.Reader) (Message, error) {
	sd := scale.Decoder{Reader: r}
	msg := new(BlockAnnounceMessage)
	_, err := sd.Decode(msg)
	return msg, err
}

// BlockAnnounceHandshake is exchanged by nodes that are beginning the BlockAnnounce protocol
type BlockAnnounceHandshake struct {
	Roles           byte
	BestBlockNumber uint64
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

// IDString ...
func (hs *BlockAnnounceHandshake) IDString() string {
	return ""
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
		BestBlockNumber: latestBlock.Number.Uint64(),
		BestBlockHash:   latestBlock.Hash(),
		GenesisHash:     s.blockState.GenesisHash(),
	}, nil
}

func (s *Service) validateBlockAnnounceHandshake(hs Handshake) error {
	if _, ok := hs.(*BlockAnnounceHandshake); !ok {
		return errors.New("invalid handshake type")
	}

	if hs.(*BlockAnnounceHandshake).GenesisHash != s.blockState.GenesisHash() {
		return errors.New("genesis hash mismatch")
	}

	return nil
}

// handleBlockAnnounceMessage handles BlockAnnounce messages
// if some more blocks are required to sync the announced block, the node will open a sync stream
// with its peer and send a BlockRequest message
func (s *Service) handleBlockAnnounceMessage(peer peer.ID, msg Message) error {
	if an, ok := msg.(*BlockAnnounceMessage); ok {
		req := s.syncer.HandleBlockAnnounce(an)
		if req != nil {
			s.requestTracker.addRequestedBlockID(req.ID)
			err := s.host.send(peer, syncID, req)
			if err != nil {
				logger.Error("failed to send BlockRequest message", "peer", peer)
			}
		}
	}

	return nil
}
