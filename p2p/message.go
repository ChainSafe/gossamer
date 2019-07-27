package p2p

import (
	scale "github.com/ChainSafe/gossamer/codec"
	common "github.com/ChainSafe/gossamer/common"
)

type RawMessage []byte

type StatusMessage struct {
	ProtocolVersion int32
	Roles           byte
	BestBlockNumber int64
	BestBlockHash   common.Hash
	GenesisHash     common.Hash
	ChainStatus     []byte
}

func (m RawMessage) Decode() (res interface{}, messageType byte, err error) {
	messageType = []byte(m)[0]

	switch messageType {
	case byte(0):
		res, err = scale.Decode(m[1:], new(StatusMessage))
	}

	return res, messageType, err
}
