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
	"errors"
	"fmt"
	"io"

	common "github.com/ChainSafe/gossamer/common"
	scale "github.com/ChainSafe/gossamer/codec"
)

const (
	StatusMsg = iota
	BlockRequestMsg
	BlockResponseMsg
	BlockAnnounceMsg
	TransactionMsg
	ConsensusMsg
	RemoteCallRequest
	RemoteCallResponse
	RemoteReadRequest
	RemoteReadResponse
	RemoteHeaderRequest
	RemoteHeaderResponse
	RemoteChangesRequest
	RemoteChangesResponse
	ChainSpecificMsg = 255
)

type Message interface{
	Encode() ([]byte, error)
	Decode([]byte) (error)
	String() string
}

// DecodeMessage accepts a raw message including the type indicator byte and decodes it to its specific message type 
func DecodeMessage(r io.Reader) (m Message, err error) {
	msgType := make([]byte, 1)
	r.Read(msgType)
	sd := scale.Decoder{Reader: r}

	switch msgType[0] {
	case StatusMsg:
		m = new(StatusMessage)
		_, err = sd.Decode(m)
	case BlockRequestMsg:
		m = new(BlockRequestMessage)
		_, err = sd.Decode(m)
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

func (sm *StatusMessage) String() string {
	return fmt.Sprintf("ProtocolVersion=%d MinSupportedVersion=%d Roles=%d BestBlockNumber=%d BestBlockHash=0x%x GenesisHash=0x%x ChainStatus=0x%x",
		sm.ProtocolVersion,
		sm.MinSupportedVersion,
		sm.Roles,
		sm.BestBlockNumber,
		sm.BestBlockHash,
		sm.GenesisHash,
		sm.ChainStatus)
}

func (sm *StatusMessage) Encode() ([]byte, error) {
    return scale.Encode(sm)
}

// Decodes the message into a StatusMessage
func (sm *StatusMessage) Decode(msg []byte) (error) {
	dec, err := scale.Decode(msg, sm)
	sm = dec.(*StatusMessage)
	return err
}

type BlockRequestMessage struct {
	Id            uint32
	RequestedData byte
	StartingBlock []byte // first byte 0 = block hash (32 byte), first byte 1 = block number (int64)
	EndBlockHash  common.Hash // optional 
	Direction     byte
	Max           uint32 // optional
}

func (bm *BlockRequestMessage) String() string {
	return fmt.Sprintf("Id=%d RequestedData=%d StartingBlock=0x%x EndBlockHash=0x%x Direction=%d Max=%d",
		bm.Id,
		bm.RequestedData,
		bm.StartingBlock,
		bm.EndBlockHash,
		bm.Direction,
		bm.Max)
}

func (bm *BlockRequestMessage) Encode() ([]byte, error) {
	return scale.Encode(bm)
}

// Decodes the message into a BlockRequestMessage
func (bm *BlockRequestMessage) Decode(msg []byte) (error) {
	dec, err := scale.Decode(msg, bm)
	bm = dec.(*BlockRequestMessage)
	return err
}