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
	"fmt"
	"io"

	common "github.com/ChainSafe/gossamer/common"
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

func (sm *StatusMessage) Encode(w io.Writer) (length int, err error) {
	err = writeUint32(w, sm.ProtocolVersion)
	if err != nil {
		return length, err
	}
	length += 4

	err = writeUint32(w, sm.MinSupportedVersion)
	if err != nil {
		return length, err
	}
	length += 4

	err = writeByte(w, sm.Roles)
	if err != nil {
		return length, err
	}
	length += 1	

	err = writeUint64(w, sm.BestBlockNumber)
	if err != nil {
		return length, err
	}
	length += 8

	err = writeHash(w, sm.BestBlockHash)
	if err != nil {
		return length, err
	}
	length += 32

	err = writeHash(w, sm.GenesisHash)
	if err != nil {
		return length, err
	}
	length += 32

	_, err = w.Write(sm.ChainStatus)
	length += len(sm.ChainStatus)

	return length, err
}

func writeByte(w io.Writer, in byte) (error) {
	buf := []byte{in}
	_, err := w.Write(buf)
	return err
}

func writeUint32(w io.Writer, in uint32) (error) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, in)
	_, err := w.Write(buf)
	return err
}

func writeUint64(w io.Writer, in uint64) (error) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, in)
	_, err := w.Write(buf)
	return err
}

func writeHash(w io.Writer, in common.Hash) (error) {
	_, err := w.Write(in.ToBytes())
	return err
}

// Decodes the buffer underlying the reader into a StatusMessage
// it reads up to specified length
func (sm *StatusMessage) Decode(r io.Reader, length uint64) (err error) {
	sm.ProtocolVersion, err = readUint32(r)
	if err != nil {
		return err
	}

	sm.MinSupportedVersion, err = readUint32(r)
	if err != nil {
		return err
	}

	sm.Roles, err = readByte(r)
	if err != nil {
		return err
	}

	sm.BestBlockNumber, err = readUint64(r)
	if err != nil {
		return err
	}

	sm.BestBlockHash, err = readHash(r)
	if err != nil {
		return err
	}

	sm.GenesisHash, err = readHash(r)
	if err != nil {
		return err
	}

	if length < 81 {
		return nil
	}

	buf := make([]byte, length-81)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	sm.ChainStatus = buf

	return nil
}

type BlockRequestMessage struct {
	Id            uint32
	RequestedData byte
	StartingBlock []byte // first byte 0 = block hash (32 byte), first byte 1 = block number (int64)
	EndBlockHash  common.Hash // optional 
	Direction     byte
	Max           uint32 // optional
}

func (bm *BlockRequestMessage) Encode(w io.Writer) (err error) {
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
