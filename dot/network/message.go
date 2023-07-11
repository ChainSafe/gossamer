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

// MaxBlocksInResponse is maximum number of block data a BlockResponse message can contain
const MaxBlocksInResponse = 128

type MessageType byte

// Message types for notifications protocol messages. Used internally to map message to protocol.
const (
	blockAnnounceMsgType MessageType = iota + 3
	transactionMsgType
	ConsensusMsgType
	CollationMsgType
	ValidationMsgType
)

// Message must be implemented by all network messages
type Message interface {
	Encode() ([]byte, error)
}

// NotificationsMessage must be implemented by all messages sent over a notifications protocol
type NotificationsMessage interface {
	Message
	Type() MessageType
	Hash() (common.Hash, error)
}

const (
	RequestedDataHeader        = byte(1)
	RequestedDataBody          = byte(2)
	RequestedDataReceipt       = byte(4)
	RequestedDataMessageQueue  = byte(8)
	RequestedDataJustification = byte(16)
	BootstrapRequestData       = RequestedDataHeader +
		RequestedDataBody +
		RequestedDataJustification
)

var _ Message = (*BlockRequestMessage)(nil)

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
	StartingBlock variadic.Uint32OrHash // first byte 0 = block hash (32 byte), first byte 1 = block number (uint32)
	Direction     SyncDirection         // 0 = ascending, 1 = descending
	Max           *uint32
}

// String formats a BlockRequestMessage as a string
func (bm *BlockRequestMessage) String() string {
	max := uint32(0)
	if bm.Max != nil {
		max = *bm.Max
	}
	return fmt.Sprintf("BlockRequestMessage RequestedData=%d StartingBlock=%v Direction=%d Max=%d",
		bm.RequestedData,
		bm.StartingBlock,
		bm.Direction,
		max)
}

// Encode returns the protobuf encoded BlockRequestMessage
func (bm *BlockRequestMessage) Encode() ([]byte, error) {
	var max uint32
	if bm.Max != nil {
		max = *bm.Max
	}

	msg := &pb.BlockRequest{
		Fields:    uint32(bm.RequestedData) << 24, // put byte in most significant byte of uint32
		Direction: pb.Direction(bm.Direction),
		MaxBlocks: max,
	}

	if bm.StartingBlock.IsHash() {
		hash := bm.StartingBlock.Hash()
		msg.FromBlock = &pb.BlockRequest_Hash{
			Hash: hash[:],
		}
	} else if bm.StartingBlock.IsUint32() {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, bm.StartingBlock.Uint32())
		msg.FromBlock = &pb.BlockRequest_Number{
			Number: buf,
		}
	} else {
		return nil, errInvalidStartingBlockType
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
		startingBlock *variadic.Uint32OrHash
		max           *uint32
	)

	switch from := msg.FromBlock.(type) {
	case *pb.BlockRequest_Hash:
		startingBlock, err = variadic.NewUint32OrHash(common.BytesToHash(from.Hash))
	case *pb.BlockRequest_Number:
		if len(from.Number) != 4 {
			return fmt.Errorf("%w expected 4 bytes, got %d bytes", errBlockRequestFromNumberInvalid, len(from.Number))
		}

		number := binary.LittleEndian.Uint32(from.Number)
		startingBlock, err = variadic.NewUint32OrHash(number)
	default:
		err = errors.New("invalid StartingBlock")
	}

	if err != nil {
		return err
	}

	if msg.MaxBlocks != 0 {
		max = &msg.MaxBlocks
	} else {
		max = nil
	}

	bm.RequestedData = byte(msg.Fields >> 24)
	bm.StartingBlock = *startingBlock
	bm.Direction = SyncDirection(byte(msg.Direction))
	bm.Max = max

	return nil
}

var _ ResponseMessage = (*BlockResponseMessage)(nil)

// BlockResponseMessage is sent in response to a BlockRequestMessage
type BlockResponseMessage struct {
	BlockData []*types.BlockData
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

// Type returns ConsensusMsgType
func (*ConsensusMessage) Type() MessageType {
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
func (cm *ConsensusMessage) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := cm.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}
	return common.Blake2bHash(encMsg)
}

func NewBlockRequest(startingBlock variadic.Uint32OrHash, amount uint32,
	requestedData byte, direction SyncDirection) *BlockRequestMessage {
	return &BlockRequestMessage{
		RequestedData: requestedData,
		StartingBlock: startingBlock,
		Direction:     direction,
		Max:           &amount,
	}
}

func NewAscendingBlockRequests(startNumber, targetNumber uint, requestedData byte) []*BlockRequestMessage {
	if startNumber > targetNumber {
		return []*BlockRequestMessage{}
	}

	diff := targetNumber - (startNumber - 1)

	// start and end block are the same, just request 1 block
	if diff == 0 {
		return []*BlockRequestMessage{
			NewBlockRequest(*variadic.MustNewUint32OrHash(uint32(startNumber)), 1, requestedData, Ascending),
		}
	}

	numRequests := diff / MaxBlocksInResponse
	// we should check if the diff is in the maxResponseSize bounds
	// otherwise we should increase the numRequests by one, take this
	// example, we want to sync from 0 to 259, the diff is 259
	// then the num of requests is 2 (uint(259)/uint(128)) however two requests will
	// retrieve only 256 blocks (each request can retrieve a max of 128 blocks), so we should
	// create one more request to retrieve those missing blocks, 3 in this example.
	missingBlocks := diff % MaxBlocksInResponse
	if missingBlocks != 0 {
		numRequests++
	}

	reqs := make([]*BlockRequestMessage, numRequests)
	for i := uint(0); i < numRequests; i++ {
		max := uint32(MaxBlocksInResponse)

		lastIteration := numRequests - 1
		if i == lastIteration && missingBlocks != 0 {
			max = uint32(missingBlocks)
		}

		start := variadic.MustNewUint32OrHash(startNumber)
		reqs[i] = NewBlockRequest(*start, max, requestedData, Ascending)
		startNumber += uint(max)
	}

	return reqs
}
