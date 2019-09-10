package babe

import (
	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/runtime"
)

type VrfPublicKey [32]byte
type VrfPrivateKey [64]byte

type BabeSession struct {
	vrfPublicKey  VrfPublicKey
	vrfPrivateKey VrfPrivateKey
	rt            *runtime.Runtime

	// currentEpoch uint64
	// currentSlot  uint64

	// TODO: TransactionQueue
}

func NewBabeSession(pubkey VrfPublicKey, privkey VrfPrivateKey, rt *runtime.Runtime) *BabeSession {
	return &BabeSession{
		vrfPublicKey:  pubkey,
		vrfPrivateKey: privkey,
		rt:            rt,
	}
}

type BabeConfiguration struct {
	SlotDuration         uint64
	C1                   uint64 // (1-(c1/c2)) is the probability of a slot being empty
	C2                   uint64
	MedianRequiredBlocks uint64
}

// gets number of slots in epoch
func (b *BabeSession) startupData() (*BabeConfiguration, error) {
	ret, err := b.rt.Exec("BabeApi_startup_data", 1, 0)
	if err != nil {
		return nil, err
	}

	bc := new(BabeConfiguration)
	_, err = scale.Decode(ret, bc)
	return bc, err
}

// TODO: change to Schnorrkel public key
type AuthorityId [32]byte

type Epoch struct {
	EpochIndex     uint64
	StartSlot      uint64
	Duration       uint64
	Authorities    AuthorityId // Schnorrkel public key of authority
	Randomness     byte
	SecondarySlots bool
}

// gets number of slots in epoch
func (b *BabeSession) epoch() (*Epoch, error) {
	ret, err := b.rt.Exec("BabeApi_epoch", 1, 0)
	if err != nil {
		return nil, err
	}

	e := new(Epoch)
	_, err = scale.Decode(ret, e)
	return e, err
}
