// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"encoding/binary"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	pb "github.com/ChainSafe/gossamer/dot/network/proto"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Message types for notifications protocol messages. Used internally to map message to protocol.
const (
	BlockAnnounceMsgType byte = 3
	TransactionMsgType   byte = 4
	ConsensusMsgType     byte = 5
)

// Message must be implemented by all network messages
type Message interface {
	SubProtocol() string
	Encode() ([]byte, error)
	Decode([]byte) error
	String() string
}

// NotificationsMessage must be implemented by all messages sent over a notifications protocol
type NotificationsMessage interface {
	Message
	Type() byte
	Hash() common.Hash
	IsHandshake() bool
}

//nolint:revive
const (
	RequestedDataHeader        = byte(1)
	RequestedDataBody          = byte(2)
	RequestedDataReceipt       = byte(4)
	RequestedDataMessageQueue  = byte(8)
	RequestedDataJustification = byte(16)
)

var _ Message = &BlockRequestMessage{}

// SyncDirection is the direction of data in a block response
type SyncDirection byte

const (
	// Ascending is used when block response data is in ascending order (ie parent to child)
	Ascending SyncDirection = iota

	// Descending is used when block response data is in descending order (ie child to parent)
	Descending
)

func (sd SyncDirection) String() string {
	switch sd {
	case Ascending:
		return "ascending"
	case Descending:
		return "descending"
	default:
		return "invalid"
	}
}

// BlockRequestMessage is sent to request some blocks from a peer
type BlockRequestMessage struct {
	RequestedData byte
	StartingBlock variadic.Uint64OrHash // first byte 0 = block hash (32 byte), first byte 1 = block number (int64)
	EndBlockHash  *common.Hash
	Direction     SyncDirection // 0 = ascending, 1 = descending
	Max           *uint32
}

// SubProtocol returns the sync sub-protocol
func (bm *BlockRequestMessage) SubProtocol() string {
	return syncID
}

// String formats a BlockRequestMessage as a string
func (bm *BlockRequestMessage) String() string {
	hash := common.Hash{}
	max := uint32(0)
	if bm.EndBlockHash != nil {
		hash = *bm.EndBlockHash
	}
	if bm.Max != nil {
		max = *bm.Max
	}
	return fmt.Sprintf("BlockRequestMessage RequestedData=%d StartingBlock=%v EndBlockHash=%s Direction=%d Max=%d",
		bm.RequestedData,
		bm.StartingBlock,
		hash.String(),
		bm.Direction,
		max)
}

// Encode returns the protobuf encoded BlockRequestMessage
func (bm *BlockRequestMessage) Encode() ([]byte, error) {
	var (
		toBlock []byte
		max     uint32
	)

	if bm.EndBlockHash != nil {
		hash := bm.EndBlockHash
		toBlock = hash[:]
	}

	if bm.Max != nil {
		max = *bm.Max
	}

	msg := &pb.BlockRequest{
		Fields:    uint32(bm.RequestedData) << 24, // put byte in most significant byte of uint32
		ToBlock:   toBlock,
		Direction: pb.Direction(bm.Direction),
		MaxBlocks: max,
	}

	if bm.StartingBlock.IsHash() {
		hash := bm.StartingBlock.Hash()
		msg.FromBlock = &pb.BlockRequest_Hash{
			Hash: hash[:],
		}
	} else if bm.StartingBlock.IsUint64() {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, bm.StartingBlock.Uint64())
		msg.FromBlock = &pb.BlockRequest_Number{
			Number: buf,
		}
	} else {
		return nil, errors.New("invalid StartingBlock in messsage")
	}

	return proto.Marshal(msg)
}

// Decode decodes the protobuf encoded input to a BlockRequestMessage
func (bm *BlockRequestMessage) Decode(in []byte) error {
	msg := &pb.BlockRequest{}
	err := proto.Unmarshal(in, msg)
	if err != nil {
		return err
	}

	var (
		startingBlock *variadic.Uint64OrHash
		endBlockHash  *common.Hash
		max           *uint32
	)

	switch from := msg.FromBlock.(type) {
	case *pb.BlockRequest_Hash:
		startingBlock, err = variadic.NewUint64OrHash(common.BytesToHash(from.Hash))
	case *pb.BlockRequest_Number:
		// TODO: we are receiving block requests w/ 4-byte From field; this should probably be
		// 4-bytes as it represents a block number which is uint32 (#1854)
		if len(from.Number) != 8 {
			return errors.New("invalid BlockResponseMessage.From; uint64 is not 8 bytes")
		}
		startingBlock, err = variadic.NewUint64OrHash(binary.LittleEndian.Uint64(from.Number))
	default:
		err = errors.New("invalid StartingBlock")
	}

	if err != nil {
		return err
	}

	if len(msg.ToBlock) != 0 {
		hash := common.NewHash(msg.ToBlock)
		endBlockHash = &hash
	} else {
		endBlockHash = nil
	}

	if msg.MaxBlocks != 0 {
		max = &msg.MaxBlocks
	} else {
		max = nil
	}

	bm.RequestedData = byte(msg.Fields >> 24)
	bm.StartingBlock = *startingBlock
	bm.EndBlockHash = endBlockHash
	bm.Direction = SyncDirection(byte(msg.Direction))
	bm.Max = max

	return nil
}

var _ Message = &BlockResponseMessage{}

// BlockResponseMessage is sent in response to a BlockRequestMessage
type BlockResponseMessage struct {
	BlockData []*types.BlockData
}

// SubProtocol returns the sync sub-protocol
func (bm *BlockResponseMessage) SubProtocol() string {
	return syncID
}

// String formats a BlockResponseMessage as a string
func (bm *BlockResponseMessage) String() string {
	if bm == nil {
		return "BlockResponseMessage=nil"
	}

	return fmt.Sprintf("BlockResponseMessage BlockData=%v", bm.BlockData)
}

// Encode returns the protobuf encoded BlockResponseMessage
func (bm *BlockResponseMessage) Encode() ([]byte, error) {
	var (
		err error
	)

	msg := &pb.BlockResponse{
		Blocks: make([]*pb.BlockData, len(bm.BlockData)),
	}

	for i, bd := range bm.BlockData {
		msg.Blocks[i], err = blockDataToProtobuf(bd)
		if err != nil {
			return nil, err
		}
	}

	return proto.Marshal(msg)
}

// Decode decodes the protobuf encoded input to a BlockResponseMessage
func (bm *BlockResponseMessage) Decode(in []byte) (err error) {
	msg := &pb.BlockResponse{}
	err = proto.Unmarshal(in, msg)
	if err != nil {
		return err
	}

	bm.BlockData = make([]*types.BlockData, len(msg.Blocks))

	for i, bd := range msg.Blocks {
		block, err := protobufToBlockData(bd)
		if err != nil {
			return err
		}
		bm.BlockData[i] = block
	}

	return nil
}

// blockDataToProtobuf converts a gossamer BlockData to a protobuf-defined BlockData
func blockDataToProtobuf(bd *types.BlockData) (*pb.BlockData, error) {
	p := &pb.BlockData{
		Hash: bd.Hash[:],
	}

	if bd.Header != nil {
		header, err := scale.Marshal(*bd.Header)
		if err != nil {
			return nil, err
		}
		p.Header = header
	}

	if bd.Body != nil {
		body := bd.Body
		exts, err := body.AsEncodedExtrinsics()
		if err != nil {
			return nil, err
		}

		p.Body = types.ExtrinsicsArrayToBytesArray(exts)
	}

	if bd.Receipt != nil {
		p.Receipt = *bd.Receipt
	}

	if bd.MessageQueue != nil {
		p.MessageQueue = *bd.MessageQueue
	}

	if bd.Justification != nil {
		p.Justification = *bd.Justification
		if len(*bd.Justification) == 0 {
			p.IsEmptyJustification = true
		}
	}

	return p, nil
}

func protobufToBlockData(pbd *pb.BlockData) (*types.BlockData, error) {
	bd := &types.BlockData{
		Hash: common.BytesToHash(pbd.Hash),
	}

	if pbd.Header != nil {
		header := types.NewEmptyHeader()
		err := scale.Unmarshal(pbd.Header, header)
		if err != nil {
			return nil, err
		}

		bd.Header = header
	}

	if pbd.Body != nil {
		body, err := types.NewBodyFromEncodedBytes(pbd.Body)
		if err != nil {
			return nil, err
		}

		bd.Body = body
	} else {
		bd.Body = nil
	}

	if pbd.Receipt != nil {
		bd.Receipt = &pbd.Receipt
	} else {
		bd.Receipt = nil
	}

	if pbd.MessageQueue != nil {
		bd.MessageQueue = &pbd.MessageQueue
	} else {
		bd.MessageQueue = nil
	}

	if pbd.Justification != nil {
		bd.Justification = &pbd.Justification
	} else {
		bd.Justification = nil
	}

	if pbd.Justification == nil && pbd.IsEmptyJustification {
		bd.Justification = &[]byte{}
	}

	return bd, nil
}

var _ NotificationsMessage = &ConsensusMessage{}

// ConsensusMessage is mostly opaque to us
type ConsensusMessage struct {
	Data []byte
}

// SubProtocol returns the empty, since consensus message sub-protocol is determined by the package using it
func (cm *ConsensusMessage) SubProtocol() string {
	return ""
}

// Type returns ConsensusMsgType
func (cm *ConsensusMessage) Type() byte {
	return ConsensusMsgType
}

// String is the string
func (cm *ConsensusMessage) String() string {
	return fmt.Sprintf("ConsensusMessage Data=%x", cm.Data)
}

// Encode encodes a block response message using SCALE
func (cm *ConsensusMessage) Encode() ([]byte, error) {
	return cm.Data, nil
}

// Decode the message into a ConsensusMessage
func (cm *ConsensusMessage) Decode(in []byte) error {
	cm.Data = in
	return nil
}

// Hash returns the Hash of ConsensusMessage
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
