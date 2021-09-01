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
	"errors"
	"fmt"
	scale2 "github.com/ChainSafe/gossamer/pkg/scale"

	pb "github.com/ChainSafe/gossamer/dot/network/proto"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/scale"

	"google.golang.org/protobuf/proto"
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

// nolint
const (
	RequestedDataHeader        = byte(1)
	RequestedDataBody          = byte(2)
	RequestedDataReceipt       = byte(4)
	RequestedDataMessageQueue  = byte(8)
	RequestedDataJustification = byte(16)
)

var _ Message = &BlockRequestMessage{}

// BlockRequestMessage is sent to request some blocks from a peer
// TODO remove optionals from this struct
type BlockRequestMessage struct {
	RequestedData byte
	StartingBlock *variadic.Uint64OrHash // first byte 0 = block hash (32 byte), first byte 1 = block number (int64)
	EndBlockHash  *optional.Hash
	Direction     byte // 0 = ascending, 1 = descending
	Max           *optional.Uint32
}

// SubProtocol returns the sync sub-protocol
func (bm *BlockRequestMessage) SubProtocol() string {
	return syncID
}

// String formats a BlockRequestMessage as a string
func (bm *BlockRequestMessage) String() string {
	return fmt.Sprintf("BlockRequestMessage RequestedData=%d StartingBlock=0x%x EndBlockHash=%s Direction=%d Max=%s",
		bm.RequestedData,
		bm.StartingBlock,
		bm.EndBlockHash.String(),
		bm.Direction,
		bm.Max.String())
}

// Encode returns the protobuf encoded BlockRequestMessage
func (bm *BlockRequestMessage) Encode() ([]byte, error) {
	var (
		toBlock []byte
		max     uint32
	)

	if bm.EndBlockHash.Exists() {
		hash := bm.EndBlockHash.Value()
		toBlock = hash[:]
	}

	if bm.Max.Exists() {
		max = bm.Max.Value()
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
		endBlockHash  *optional.Hash
		max           *optional.Uint32
	)

	switch from := msg.FromBlock.(type) {
	case *pb.BlockRequest_Hash:
		startingBlock, err = variadic.NewUint64OrHash(common.BytesToHash(from.Hash))
	case *pb.BlockRequest_Number:
		// TODO: we are receiving block requests w/ 4-byte From field; did the format change?
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
		endBlockHash = optional.NewHash(true, common.BytesToHash(msg.ToBlock))
	} else {
		endBlockHash = optional.NewHash(false, common.Hash{})
	}

	if msg.MaxBlocks != 0 {
		max = optional.NewUint32(true, msg.MaxBlocks)
	} else {
		max = optional.NewUint32(false, 0)
	}

	bm.RequestedData = byte(msg.Fields >> 24)
	bm.StartingBlock = startingBlock
	bm.EndBlockHash = endBlockHash
	bm.Direction = byte(msg.Direction)
	bm.Max = max

	return nil
}

var _ Message = &BlockResponseMessage{}
var _ Message = &BlockResponseMessageNew{}

// BlockResponseMessage is sent in response to a BlockRequestMessage
type BlockResponseMessage struct {
	BlockData []*types.BlockData
}

type BlockResponseMessageNew struct {
	BlockData []*types.BlockDataVdt
}

func (bm *BlockResponseMessageNew) getStartAndEnd() (int64, int64, error) {
	if len(bm.BlockData) == 0 {
		return 0, 0, errors.New("no BlockData in BlockResponseMessage")
	}

	if startExists := bm.BlockData[0].Header.Exists(); !startExists {
		return 0, 0, errors.New("first BlockData in BlockResponseMessage does not contain header")
	}

	if endExists := bm.BlockData[len(bm.BlockData)-1].Header.Exists(); !endExists {
		return 0, 0, errors.New("last BlockData in BlockResponseMessage does not contain header")
	}

	return bm.BlockData[0].Header.Number.Int64(), bm.BlockData[len(bm.BlockData)-1].Header.Number.Int64(), nil
}

func (bm *BlockResponseMessage) getStartAndEnd() (int64, int64, error) {
	if len(bm.BlockData) == 0 {
		return 0, 0, errors.New("no BlockData in BlockResponseMessage")
	}

	if startExists := bm.BlockData[0].Header.Exists(); !startExists {
		return 0, 0, errors.New("first BlockData in BlockResponseMessage does not contain header")
	}

	if endExists := bm.BlockData[len(bm.BlockData)-1].Header.Exists(); !endExists {
		return 0, 0, errors.New("last BlockData in BlockResponseMessage does not contain header")
	}

	return bm.BlockData[0].Header.Value().Number.Int64(), bm.BlockData[len(bm.BlockData)-1].Header.Value().Number.Int64(), nil
}

func (bm *BlockResponseMessageNew) SubProtocol() string {
	return syncID
}

// SubProtocol returns the sync sub-protocol
func (bm *BlockResponseMessage) SubProtocol() string {
	return syncID
}

func (bm *BlockResponseMessageNew) String() string {
	if bm == nil {
		return "BlockResponseMessage=nil"
	}

	return fmt.Sprintf("BlockResponseMessage BlockData=%v", bm.BlockData)
}

// String formats a BlockResponseMessage as a string
func (bm *BlockResponseMessage) String() string {
	if bm == nil {
		return "BlockResponseMessage=nil"
	}

	return fmt.Sprintf("BlockResponseMessage BlockData=%v", bm.BlockData)
}

func (bm *BlockResponseMessageNew) Encode() ([]byte, error) {
	var (
		err error
	)

	msg := &pb.BlockResponse{
		Blocks: make([]*pb.BlockData, len(bm.BlockData)),
	}

	for i, bd := range bm.BlockData {
		msg.Blocks[i], err = blockDataToProtobufNew(bd)
		if err != nil {
			return nil, err
		}
	}

	return proto.Marshal(msg)
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

func (bm *BlockResponseMessageNew) Decode(in []byte) (err error) {
	msg := &pb.BlockResponse{}
	err = proto.Unmarshal(in, msg)
	if err != nil {
		return err
	}

	bm.BlockData = make([]*types.BlockDataVdt, len(msg.Blocks))

	for i, bd := range msg.Blocks {
		block, err := protobufToBlockDataNew(bd)
		if err != nil {
			return err
		}
		bm.BlockData[i] = block
	}

	return nil
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
		bm.BlockData[i], err = protobufToBlockData(bd)
		if err != nil {
			return err
		}
	}

	return nil
}

func blockDataToProtobufNew(bd *types.BlockDataVdt) (*pb.BlockData, error) {
	p := &pb.BlockData{
		Hash: bd.Hash[:],
	}

	if bd.Header != nil {
		header, err := scale2.Marshal(*bd.Header)
		if err != nil {
			return nil, err
		}
		p.Header = header
	}

	if bd.Body != nil {
		body := bd.Body
		exts, err := body.AsEncodedExtrinsicsNew()
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

// blockDataToProtobuf converts a gossamer BlockData to a protobuf-defined BlockData
func blockDataToProtobuf(bd *types.BlockData) (*pb.BlockData, error) {
	p := &pb.BlockData{
		Hash: bd.Hash[:],
	}

	if bd.Header != nil && bd.Header.Exists() {
		header, err := types.NewHeaderFromOptional(bd.Header)
		if err != nil {
			return nil, err
		}

		p.Header, err = header.Encode()
		if err != nil {
			return nil, err
		}
	}

	if bd.Body != nil && bd.Body.Exists() {
		body := types.Body(bd.Body.Value())
		exts, err := body.AsEncodedExtrinsics()
		if err != nil {
			return nil, err
		}

		p.Body = types.ExtrinsicsArrayToBytesArray(exts)
	}

	if bd.Receipt != nil && bd.Receipt.Exists() {
		p.Receipt = bd.Receipt.Value()
	}

	if bd.MessageQueue != nil && bd.MessageQueue.Exists() {
		p.MessageQueue = bd.MessageQueue.Value()
	}

	if bd.Justification != nil && bd.Justification.Exists() {
		p.Justification = bd.Justification.Value()
		if len(bd.Justification.Value()) == 0 {
			p.IsEmptyJustification = true
		}
	}

	return p, nil
}

func protobufToBlockDataNew(pbd *pb.BlockData) (*types.BlockDataVdt, error) {
	bd := &types.BlockDataVdt{
		Hash:   common.BytesToHash(pbd.Hash),
		Header: types.NewEmptyHeaderVdt(),
	}

	if pbd.Header != nil {
		header := types.NewEmptyHeaderVdt()
		err := scale2.Unmarshal(pbd.Header, header)
		//header, err := scale.Decode(pbd.Header, types.NewEmptyHeader())
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

func protobufToBlockData(pbd *pb.BlockData) (*types.BlockData, error) {
	bd := &types.BlockData{
		Hash: common.BytesToHash(pbd.Hash),
	}

	if pbd.Header != nil {
		header, err := scale.Decode(pbd.Header, types.NewEmptyHeader())
		if err != nil {
			return nil, err
		}

		bd.Header = header.(*types.Header).AsOptional()
	}

	if pbd.Body != nil {
		body, err := types.NewBodyFromEncodedBytes(pbd.Body)
		if err != nil {
			return nil, err
		}

		bd.Body = body.AsOptional()
	} else {
		bd.Body = optional.NewBody(false, nil)
	}

	if pbd.Receipt != nil {
		bd.Receipt = optional.NewBytes(true, pbd.Receipt)
	} else {
		bd.Receipt = optional.NewBytes(false, nil)
	}

	if pbd.MessageQueue != nil {
		bd.MessageQueue = optional.NewBytes(true, pbd.MessageQueue)
	} else {
		bd.MessageQueue = optional.NewBytes(false, nil)
	}

	if pbd.Justification != nil {
		bd.Justification = optional.NewBytes(true, pbd.Justification)
	} else {
		bd.Justification = optional.NewBytes(false, nil)
	}

	if pbd.Justification == nil && pbd.IsEmptyJustification {
		bd.Justification = optional.NewBytes(true, []byte{})
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
