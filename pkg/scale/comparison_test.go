package scale

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// BabePrimaryPreDigest as defined in Polkadot RE Spec, definition 5.10 in section 5.1.4
type BabePrimaryPreDigest struct {
	AuthorityIndex uint32
	SlotNumber     uint64
	VrfOutput      [sr25519.VrfOutputLength]byte
	VrfProof       [sr25519.VrfProofLength]byte
}

// Encode performs SCALE encoding of a BABEPrimaryPreDigest
func (d *BabePrimaryPreDigest) Encode() []byte {
	enc := []byte{byte(1)}

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, d.AuthorityIndex)
	enc = append(enc, buf...)

	buf = make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, d.SlotNumber)
	enc = append(enc, buf...)
	enc = append(enc, d.VrfOutput[:]...)
	enc = append(enc, d.VrfProof[:]...)
	return enc
}

func TestOldVsNewEncoding(t *testing.T) {
	bh := &BabePrimaryPreDigest{
		VrfOutput:      [sr25519.VrfOutputLength]byte{0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152, 85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40, 246, 76, 38, 235, 204, 43, 31, 179, 28},
		VrfProof:       [sr25519.VrfProofLength]byte{120, 23, 235, 159, 115, 122, 207, 206, 123, 232, 75, 243, 115, 255, 131, 181, 219, 241, 200, 206, 21, 22, 238, 16, 68, 49, 86, 99, 76, 139, 39, 0, 102, 106, 181, 136, 97, 141, 187, 1, 234, 183, 241, 28, 27, 229, 133, 8, 32, 246, 245, 206, 199, 142, 134, 124, 226, 217, 95, 30, 176, 246, 5, 3},
		AuthorityIndex: 17,
		SlotNumber:     420,
	}
	oldEncode := bh.Encode()
	newEncode, err := Marshal(bh)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}
	if !reflect.DeepEqual(oldEncode, newEncode) {
		t.Errorf("encodeState.encodeStruct() = %v, want %v", oldEncode, newEncode)
	}
}

// ChangesTrieRootDigest contains the root of the changes trie at a given block, if the runtime supports it.
type ChangesTrieRootDigest struct {
	Hash common.Hash
}

func (ctrd ChangesTrieRootDigest) Index() uint {
	return 2
}

// PreRuntimeDigest contains messages from the consensus engine to the runtime.
type PreRuntimeDigest struct {
	ConsensusEngineID types.ConsensusEngineID
	Data              []byte
}

func (prd PreRuntimeDigest) Index() uint {
	return 6
}

// ConsensusDigest contains messages from the runtime to the consensus engine.
type ConsensusDigest struct {
	ConsensusEngineID types.ConsensusEngineID
	Data              []byte
}

func (prd ConsensusDigest) Index() uint {
	return 4
}

// SealDigest contains the seal or signature. This is only used by native code.
type SealDigest struct {
	ConsensusEngineID types.ConsensusEngineID
	Data              []byte
}

func (prd SealDigest) Index() uint {
	return 5
}

func TestOldVsNewEncoding2(t *testing.T) {
	oldDigest := types.Digest{
		&types.ChangesTrieRootDigest{
			Hash: common.Hash{0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152, 85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40, 246, 76, 38, 235, 204, 43, 31, 179, 28},
		},
		&types.PreRuntimeDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              []byte{1, 3, 5, 7},
		},
		&types.ConsensusDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              []byte{1, 3, 5, 7},
		},
		&types.SealDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              []byte{1, 3, 5, 7},
		},
	}
	oldEncode, err := oldDigest.Encode()
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}

	newDigest := VaryingDataType{
		ChangesTrieRootDigest{
			Hash: common.Hash{0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152, 85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40, 246, 76, 38, 235, 204, 43, 31, 179, 28},
		},
		PreRuntimeDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              []byte{1, 3, 5, 7},
		},
		ConsensusDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              []byte{1, 3, 5, 7},
		},
		SealDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              []byte{1, 3, 5, 7},
		},
	}

	newEncode, err := Marshal(newDigest)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}
	if !reflect.DeepEqual(oldEncode, newEncode) {
		t.Errorf("encodeState.encodeStruct() = %v, want %v", oldEncode, newEncode)
	}
}
