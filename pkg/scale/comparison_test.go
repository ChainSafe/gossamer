package scale

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

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

func TestOldVsNewEncoding(t *testing.T) {
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

	type Digests VaryingDataType
	err = RegisterVaryingDataType(Digests{}, ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{})
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}
	newDigest := Digests{
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

	var decoded Digests
	err = Unmarshal(newEncode, &decoded)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(decoded, newDigest) {
		t.Errorf("Unmarshal() = %v, want %v", decoded, newDigest)
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	for _, tt := range allTests {
		dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
		if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
			b.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
	}
}

// func BenchmarkDecode(b *testing.B) {
// 	for _, tt := range variableWidthIntegerTests {
// 		dst := reflect.New(reflect.TypeOf(tt.in)).Interface()
// 		fmt.Printf("%v %T\n", dst, dst)
// 		// if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
// 		// 	b.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
// 		// 	return
// 		// }
// 		_, err := oldScale.Decode(tt.want, dst)
// 		if err != nil {
// 			b.Errorf("%v", err)
// 			return
// 		}
// 	}
// }
