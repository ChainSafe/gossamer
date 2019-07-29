package p2p

import (
	"encoding/binary"
	"io"

	common "github.com/ChainSafe/gossamer/common"
)

type StatusMessage struct {
	ProtocolVersion uint32
	MinSupportedVersion uint32
	Roles           byte
	BestBlockNumber uint64
	BestBlockHash   common.Hash
	GenesisHash     common.Hash
	ChainStatus     []byte
}

func (sm *StatusMessage) Decode(r io.Reader, length uint64) (err error) {
	buf := make([]byte, 4)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	sm.ProtocolVersion = binary.LittleEndian.Uint32(buf)

	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	sm.MinSupportedVersion = binary.LittleEndian.Uint32(buf)

	buf = make([]byte, 1)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	sm.Roles = buf[0]

	buf = make([]byte, 8)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	sm.BestBlockNumber = binary.LittleEndian.Uint64(buf)

	buf = make([]byte, 32)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	sm.BestBlockHash = common.NewHash(buf)

	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	sm.GenesisHash = common.NewHash(buf)

	if length < 81 {
		return nil
	}

	buf = make([]byte, length-81)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	sm.ChainStatus = buf

	return nil
}