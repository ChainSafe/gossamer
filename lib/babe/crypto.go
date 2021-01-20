package babe

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/gtank/merlin"
)

var babe_vrf_prefix = []byte("substrate-babe-vrf")

// the code in this file is based off https://github.com/paritytech/substrate/blob/89275433863532d797318b75bb5321af098fea7c/primitives/consensus/babe/src/lib.rs#L93

func makeTranscript(randomness [types.RandomnessLength]byte, slot, epoch uint64) *merlin.Transcript {
	t := merlin.NewTranscript(string(types.BabeEngineID[:]))
	t = crypto.AppendUint64(t, []byte("slot number"), slot)
	t = crypto.AppendUint64(t, []byte("current epoch"), epoch)
	t.AppendMessage([]byte("chain randomness"), randomness[:])
	return t
}

func makeTranscriptData(randomness [types.RandomnessLength]byte, slot, epoch uint64) *crypto.VRFTranscriptData {
	return &crypto.VRFTranscriptData{
		Label: string(types.BabeEngineID[:]),
		Items: map[string]*crypto.VRFTranscriptValue{
			"slot number":      &crypto.VRFTranscriptValue{Uint64: &slot},
			"current epoch":    &crypto.VRFTranscriptValue{Uint64: &epoch},
			"chain randomness": &crypto.VRFTranscriptValue{Bytes: randomness[:]},
		},
	}
}

// https://github.com/paritytech/substrate/blob/master/client/consensus/babe/src/authorship.rs#L239
func claimPrimarySlot(randomness [types.RandomnessLength]byte,
	slot, epoch uint64,
	threshold *big.Int,
	keypair *sr25519.Keypair,
) (*VrfOutputAndProof, error) {
	transcript := makeTranscript(randomness, slot, epoch)
	//transcriptData :=  makeTranscriptData(randomness, slot, epoch)
	transcript2 := makeTranscript(randomness, slot, epoch)

	out, proof, err := keypair.VrfSign(transcript)
	if err != nil {
		return nil, err
	}

	inout := sr25519.AttachInput(out, keypair.Public().(*sr25519.PublicKey), transcript2)
	res := sr25519.MakeBytes(inout, 16, babe_vrf_prefix)

	// TODO: uint128 compare

	return &VrfOutputAndProof{
		output: out,
		proof:  proof,
	}, nil
}

func verifySlotClaim(pub *sr25519.PublicKey) {

}

func checkPrimaryThreshold(randomness [types.RandomnessLength]byte,
	slot, epoch uint64,
	output [sr25519.VrfOutputLength]byte,
	threshold *big.Int,
	pub *sr25519.PublicKey,
) bool {
	t := makeTranscript(randomness, slot, epoch)
	inout := sr25519.AttachInput(output, pub, t)
	res := sr25519.MakeBytes(inout, 16, babe_vrf_prefix)
	// TODO: uint128 comparison
	return false
}
