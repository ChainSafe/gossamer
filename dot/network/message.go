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
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
)

// Message types for notifications protocol messages. Used internally to map message to protocol.
const (
	BlockAnnounceMsgType byte = 3
	TransactionMsgType   byte = 4
	ConsensusMsgType     byte = 5
)

// Message must be implemented by all network messages
type Message interface {
	Encode() ([]byte, error)
	Decode(io.Reader) error
	String() string
}

// NotificationsMessage must be implemented by all messages sent over a notifications protocol
type NotificationsMessage interface {
	Message
	Type() byte
	Hash() common.Hash
	IsHandshake() bool
}

const (
	RequestedDataHeader        = byte(1)
	RequestedDataBody          = byte(2)
	RequestedDataReceipt       = byte(4)
	RequestedDataMessageQueue  = byte(8)
	RequestedDataJustification = byte(16)
)

var _ Message = &BlockRequestMessage{}

// BlockRequestMessage is sent to request some blocks from a peer
type BlockRequestMessage struct {
	RequestedData byte
	StartingBlock *variadic.Uint64OrHash // first byte 0 = block hash (32 byte), first byte 1 = block number (int64)
	EndBlockHash  *optional.Hash
	Direction     byte // 0 = ascending, 1 = descending
	Max           *optional.Uint32
}

// String formats a BlockRequestMessage as a string
func (bm *BlockRequestMessage) String() string {
	return fmt.Sprintf("BlockRequestMessage RequestedData=%d StartingBlock=0x%x EndBlockHash=0x%s Direction=%d Max=%s",
		bm.RequestedData,
		bm.StartingBlock,
		bm.EndBlockHash.String(),
		bm.Direction,
		bm.Max.String())
}

// Encode encodes a block request message using SCALE
func (bm *BlockRequestMessage) Encode() ([]byte, error) {
	encMsg := []byte{bm.RequestedData}

	startingBlockArray, err := bm.StartingBlock.Encode()
	if err != nil || len(startingBlockArray) == 0 {
		return nil, fmt.Errorf("invalid BlockRequestMessage")
	}
	encMsg = append(encMsg, startingBlockArray...)

	if bm.EndBlockHash == nil || !bm.EndBlockHash.Exists() {
		encMsg = append(encMsg, []byte{0, 0}...)
	} else {
		val := bm.EndBlockHash.Value()
		encMsg = append(encMsg, append([]byte{1}, val[:]...)...)
	}

	encMsg = append(encMsg, bm.Direction)

	if !bm.Max.Exists() {
		encMsg = append(encMsg, []byte{0, 0}...)
	} else {
		max := make([]byte, 4)
		binary.LittleEndian.PutUint32(max, bm.Max.Value())
		encMsg = append(encMsg, append([]byte{1}, max...)...)
	}

	return encMsg, nil
}

// Decode the message into a BlockRequestMessage
func (bm *BlockRequestMessage) Decode(r io.Reader) error {
	var err error

	bm.RequestedData, err = common.ReadByte(r)
	if err != nil {
		return err
	}

	bm.StartingBlock = &variadic.Uint64OrHash{}
	err = bm.StartingBlock.Decode(r)
	if err != nil {
		return err
	}

	// EndBlockHash is an optional type, if next byte is 0 it doesn't exist
	endBlockHashExists, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	// if endBlockHash was None, then just set Direction and Max
	if endBlockHashExists == 0 {
		bm.EndBlockHash = optional.NewHash(false, common.Hash{})
	} else {
		var endBlockHash common.Hash
		endBlockHash, err = common.ReadHash(r)
		if err != nil {
			return err
		}
		bm.EndBlockHash = optional.NewHash(true, endBlockHash)
	}
	dir, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	bm.Direction = dir

	// Max is an optional type, if next byte is 0 it doesn't exist
	maxExists, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	if maxExists == 0 {
		bm.Max = optional.NewUint32(false, 0)
	} else {
		max, err := common.ReadUint32(r)
		if err != nil {
			return err
		}
		bm.Max = optional.NewUint32(true, max)
	}

	return nil
}

var _ Message = &BlockResponseMessage{}

// BlockResponseMessage is sent in response to a BlockRequestMessage
type BlockResponseMessage struct {
	BlockData []*types.BlockData
}

// String formats a BlockResponseMessage as a string
func (bm *BlockResponseMessage) String() string {
	return fmt.Sprintf("BlockResponseMessage BlockData=%v", bm.BlockData)
}

// Encode encodes a block response message using SCALE
func (bm *BlockResponseMessage) Encode() ([]byte, error) {
	return types.EncodeBlockDataArray(bm.BlockData)
}

// Decode the message into a BlockResponseMessage
func (bm *BlockResponseMessage) Decode(r io.Reader) (err error) {
	bm.BlockData, err = types.DecodeBlockDataArray(r)
	return err
}

var _ NotificationsMessage = &ConsensusMessage{}

// ConsensusMessage is mostly opaque to us
type ConsensusMessage struct {
	// Identifies consensus engine.
	ConsensusEngineID types.ConsensusEngineID
	// Message payload.
	Data []byte
}

// Type returns ConsensusMsgType
func (cm *ConsensusMessage) Type() byte {
	return ConsensusMsgType
}

// String is the string
func (cm *ConsensusMessage) String() string {
	return fmt.Sprintf("ConsensusMessage ConsensusEngineID=%d, DATA=%x", cm.ConsensusEngineID, cm.Data)
}

// Encode encodes a block response message using SCALE
func (cm *ConsensusMessage) Encode() ([]byte, error) {
	encMsg := cm.ConsensusEngineID.ToBytes()
	return append(encMsg, cm.Data...), nil
}

// Decode the message into a ConsensusMessage
func (cm *ConsensusMessage) Decode(r io.Reader) error {
	buf := make([]byte, 4)
	_, err := r.Read(buf)
	if err != nil {
		return err
	}
	cm.ConsensusEngineID = types.NewConsensusEngineID(buf)
	for {
		b, err := common.ReadByte(r)
		if err != nil {
			break
		}

		cm.Data = append(cm.Data, b)
	}

	return nil
}

// ConsensusMessage returns the Hash of ConsensusMessage
func (cm *ConsensusMessage) Hash() common.Hash {
	// scale encode each extrinsic
	encMsg, _ := cm.Encode()
	hash, _ := common.Blake2bHash(encMsg)
	return hash
}

// IsHandshake returns false
func (cm *ConsensusMessage) IsHandshake() bool {
	return false
}
