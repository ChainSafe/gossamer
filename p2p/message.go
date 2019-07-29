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
	"io"

	common "github.com/ChainSafe/gossamer/common"
)

const (
	STATUS_MESSAGE          = 0
	BLOCK_REQUEST           = 1
	BLOCK_RESPONSE          = 2
	BLOCK_ANNOUNCE          = 3
	TRANSACTION             = 4
	CONSENSUS               = 5
	REMOTE_CALL_REQUEST     = 6
	REMOTE_CALL_RESPONSE    = 7
	REMOTE_READ_REQUEST     = 8
	REMOTE_READ_RESPONSE    = 9
	REMOTE_HEADER_REQUEST   = 10
	REMOTE_HEADER_RESPONSE  = 11
	REMOTE_CHANGES_REQUEST  = 12
	REMOTE_CHANGES_RESPONSE = 13
	CHAIN_SPECIFIC          = 255
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
