package babe

import (
	"encoding/binary"

	"github.com/ChainSafe/gossamer/runtime"
	scale "github.com/ChainSafe/gossamer/codec"
)

type VrfPublicKey [32]byte
type VrfPrivateKey [64]byte

type BabeSession struct {
	vrfPublicKey  VrfPublicKey
	vrfPrivateKey VrfPrivateKey
	rt            *runtime.Runtime

	currentEpoch uint64
	currentSlot  uint64

	// TODO: TransactionQueue
}

func NewBabeSession(pubkey VrfPublicKey, privkey VrfPrivateKey, rt *runtime.Runtime) *BabeSession {
	return &BabeSession{
		vrfPublicKey:  pubkey,
		vrfPrivateKey: privkey,
		rt:            rt,
	}
}

// gets number of slots in epoch
func (b *BabeSession) startupData() (uint64, error) {
	ret, err := b.rt.Exec("BabeApi_startup_data", 1, 0)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(ret), nil
}
