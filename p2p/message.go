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

package p2p

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	scale "github.com/ChainSafe/gossamer/codec"
	common "github.com/ChainSafe/gossamer/common"
)

const (
	StatusMsgType = iota
	BlockRequestMsgType
	BlockResponseMsgType
	BlockAnnounceMsgType
	TransactionMsgType
	ConsensusMsgType
	RemoteCallRequestType
	RemoteCallResponseType
	RemoteReadRequestType
	RemoteReadResponseType
	RemoteHeaderRequestType
	RemoteHeaderResponseType
	RemoteChangesRequestType
	RemoteChangesResponseType
	ChainSpecificMsgType = 255
)

type Message interface {
	Encode() ([]byte, error)
	Decode(io.Reader) error
	String() string
}

// DecodeMessage accepts a raw message including the type indicator byte and decodes it to its specific message type
func DecodeMessage(r io.Reader) (m Message, err error) {
	msgType := make([]byte, 1)
	r.Read(msgType)

	switch msgType[0] {
	case StatusMsgType:
		m = new(StatusMessage)
		err = m.Decode(r)
	case BlockRequestMsgType:
		m = new(BlockRequestMessage)
		err = m.Decode(r)
	case BlockResponseMsgType:
		m = new(BlockResponseMessage)
		err = m.Decode(r)
	default:
		return nil, errors.New("unsupported message type")
	}

	return m, err
}

type StatusMessage struct {
	ProtocolVersion     uint32
	MinSupportedVersion uint32
	Roles               byte
	BestBlockNumber     uint64
	BestBlockHash       common.Hash
	GenesisHash         common.Hash
	ChainStatus         []byte
}

// String formats a StatusMessage as a string
func (sm *StatusMessage) String() string {
	return fmt.Sprintf("StatusMessage ProtocolVersion=%d MinSupportedVersion=%d Roles=%d BestBlockNumber=%d BestBlockHash=0x%x GenesisHash=0x%x ChainStatus=0x%x",
		sm.ProtocolVersion,
		sm.MinSupportedVersion,
		sm.Roles,
		sm.BestBlockNumber,
		sm.BestBlockHash,
		sm.GenesisHash,
		sm.ChainStatus)
}

// Encode encodes a status message using SCALE and appends the type byte to the start
func (sm *StatusMessage) Encode() ([]byte, error) {
	enc, err := scale.Encode(sm)
	if err != nil {
		return enc, err
	}
	return append([]byte{StatusMsgType}, enc...), nil
}

// Decodes the message into a StatusMessage, it assumes the type byte has been removed
func (sm *StatusMessage) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(sm)
	return err
}

type BlockRequestMessage struct {
	Id            uint64
	RequestedData byte
	StartingBlock []byte      // first byte 0 = block hash (32 byte), first byte 1 = block number (int64)
	EndBlockHash  common.Hash // optional
	Direction     byte
	Max           uint32 // optional
}

// String formats a BlockRequestMessage as a string
func (bm *BlockRequestMessage) String() string {
	return fmt.Sprintf("BlockRequestMessage Id=%d RequestedData=%d StartingBlock=0x%x EndBlockHash=0x%x Direction=%d Max=%d",
		bm.Id,
		bm.RequestedData,
		bm.StartingBlock,
		bm.EndBlockHash,
		bm.Direction,
		bm.Max)
}

// Encode encodes a block request message using SCALE and appends the type byte to the start
func (bm *BlockRequestMessage) Encode() ([]byte, error) {
	encMsg := []byte{BlockRequestMsgType}

	encId := make([]byte, 8)
	binary.LittleEndian.PutUint64(encId, bm.Id)
	encMsg = append(encMsg, encId...)

	encMsg = append(encMsg, bm.RequestedData)

	if bm.StartingBlock[0] == 1 {
		encMsg = append(encMsg, bm.StartingBlock[0])
		num := bm.StartingBlock[1:]
		if len(num) < 8 {
			num = common.AppendZeroes(num, 8)
		}
		encMsg = append(encMsg, num...)
	} else {
		encMsg = append(encMsg, bm.StartingBlock...)
	}

	if bm.EndBlockHash != [32]byte{} {
		encMsg = append(encMsg, bm.EndBlockHash.ToBytes()...)
	} else {
		encMsg = append(encMsg, 0)
	}

	encMsg = append(encMsg, bm.Direction)

	if bm.Max != 0 {
		encMax := make([]byte, 4)
		binary.LittleEndian.PutUint32(encMax, bm.Max)
		encMsg = append(encMsg, encMax...)
	} else {
		encMsg = append(encMsg, 0)
	}

	return encMsg, nil
}

// Decodes the message into a BlockRequestMessage, it assumes the type byte has been removed
func (bm *BlockRequestMessage) Decode(r io.Reader) error {
	var err error
	bm.Id, err = readUint64(r)
	if err != nil {
		return err
	}

	bm.RequestedData, err = readByte(r)
	if err != nil {
		return err
	}

	startingBlockType, err := readByte(r)
	if err != nil {
		return err
	}

	if startingBlockType == 0 {
		hash := make([]byte, 32)
		_, err = r.Read(hash)
		bm.StartingBlock = append([]byte{startingBlockType}, hash...)
	} else {
		num := make([]byte, 8)
		_, err = r.Read(num)
		bm.StartingBlock = append([]byte{startingBlockType}, num...)
	}

	endBlockHash := make([]byte, 32)
	bytesread, err := r.Read(endBlockHash)
	if err != nil {
		return err
	}

	// if endBlockHash was None, then just set Direction and Max
	if bytesread < 32 {
		bm.Direction = endBlockHash[1]
		max := binary.LittleEndian.Uint32(endBlockHash[2:6])
		bm.Max = max
	} else {
		bm.EndBlockHash = common.NewHash(endBlockHash)

		bm.Direction, err = readByte(r)
		if err != nil {
			return err
		}

		bm.Max, err = readUint32(r)
		if err != nil {
			return err
		}
	}

	return nil
}

type BlockResponseMessage struct {
	Id   uint64
	Data []byte // TODO: change this to BlockData type
}

func (bm *BlockResponseMessage) String() string {
	return fmt.Sprintf("BlockResponseMessage Id=%d Data=%x", bm.Id, bm.Data)
}

func (bm *BlockResponseMessage) Encode() ([]byte, error) {
	encMsg := []byte{BlockResponseMsgType}

	encId := make([]byte, 8)
	binary.LittleEndian.PutUint64(encId, bm.Id)
	encMsg = append(encMsg, encId...)

	return append(encMsg, bm.Data...), nil
}

func (bm *BlockResponseMessage) Decode(r io.Reader) error {
	bm.Id, err = readUint64(r)
	if err != nil {
		return err
	}

	
	return nil
}

func readByte(r io.Reader) (byte, error) {
	buf := make([]byte, 1)
	_, err := r.Read(buf)
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

func readUint32(r io.Reader) (uint32, error) {
	buf := make([]byte, 4)
	_, err := r.Read(buf)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}

func readUint64(r io.Reader) (uint64, error) {
	buf := make([]byte, 8)
	_, err := r.Read(buf)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func readHash(r io.Reader) (common.Hash, error) {
	buf := make([]byte, 32)
	_, err := r.Read(buf)
	if err != nil {
		return common.Hash{}, err
	}
	return common.NewHash(buf), nil
}
