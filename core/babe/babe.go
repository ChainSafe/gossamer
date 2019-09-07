package babe

import (
	"encoding/binary"

	"github.com/ChainSafe/gossamer/runtime"
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
func (b *BabeSession) getNumberOfSlots(epoch uint64) (uint64, error) {
	mem := b.rt.Mem()
	var input int32 = 1

	epochBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBytes, epoch)
	copy(mem[input:input+8], epochBytes)
	ret, err := b.rt.Exec("BabeApi_slot_duration", input, 8)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(ret), nil
}
