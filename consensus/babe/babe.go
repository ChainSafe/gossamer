package babe

import (
	"math/big"
	"errors"
	"github.com/ChainSafe/gossamer/runtime"
)

// BabeSession contains the VRF keys for the validator
type BabeSession struct {
	vrfPublicKey  VrfPublicKey
	vrfPrivateKey VrfPrivateKey
	rt            *runtime.Runtime

	config		*BabeConfiguration
	epochData 	*Epoch

	authorityIndex uint64
	authorityWeights []uint64

	// currentEpoch uint64
	// currentSlot  uint64

	// TODO: TransactionQueue
}

// NewBabeSession returns a new Babe session using the provided VRF keys and runtime
func NewBabeSession(pubkey VrfPublicKey, privkey VrfPrivateKey, rt *runtime.Runtime) *BabeSession {
	return &BabeSession{
		vrfPublicKey:  pubkey,
		vrfPrivateKey: privkey,
		rt:            rt,
	}
}

func (b *BabeSession) runLottery(slot uint64) (bool, error) {
	if slot < b.epochData.StartSlot {
		return false, errors.New("slot is not in this epoch")
	}

	output, err := b.vrfSign(slot)
	if err != nil {
		return false, err
	}

	threshold, err := calculateThreshold(b.config.C1, b.config.C2, b.authorityIndex, b.authorityWeights)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (b *BabeSession) vrfSign(slot uint64) ([]byte, error) {
	// TOOD: return VRF output and proof
	// sign b.epochData.Randomness and slot
	return nil, nil
}

func calculateThreshold(C1, C2, authorityIndex uint64, authorityWeights []uint64) (*big.Int, error) {
	var sum uint64 = 0
	for _, weight := range authorityWeights {
		sum += weight
	}

	var theta float64 = float64(authorityWeights[authorityIndex]) / float64(sum)

	return nil, nil

}