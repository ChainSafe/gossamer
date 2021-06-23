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
	oldDigests := types.Digest{
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
	oldEncode, err := oldDigests.Encode()
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}

	vdt, err := NewVaryingDataType(ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{})
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}
	err = vdt.Set(ChangesTrieRootDigest{
		Hash: common.Hash{0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152, 85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40, 246, 76, 38, 235, 204, 43, 31, 179, 28},
	})
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}

	newDigest := []VaryingDataType{
		mustNewVaryingDataTypeAndSet(
			ChangesTrieRootDigest{
				Hash: common.Hash{0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152, 85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40, 246, 76, 38, 235, 204, 43, 31, 179, 28},
			},
			ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{},
		),
		mustNewVaryingDataTypeAndSet(
			PreRuntimeDigest{
				ConsensusEngineID: types.BabeEngineID,
				Data:              []byte{1, 3, 5, 7},
			},
			ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{},
		),
		mustNewVaryingDataTypeAndSet(
			ConsensusDigest{
				ConsensusEngineID: types.BabeEngineID,
				Data:              []byte{1, 3, 5, 7},
			},
			ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{},
		),
		mustNewVaryingDataTypeAndSet(
			SealDigest{
				ConsensusEngineID: types.BabeEngineID,
				Data:              []byte{1, 3, 5, 7},
			},
			ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{},
		),
	}

	newEncode, err := Marshal(newDigest)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
		return
	}
	if !reflect.DeepEqual(oldEncode, newEncode) {
		t.Errorf("encodeState.encodeStruct() = %v, want %v", oldEncode, newEncode)
	}

	decoded := NewVaryingDataTypeSlice(vdt)
	err = Unmarshal(newEncode, &decoded)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	// decoded.Types
	if !reflect.DeepEqual(decoded.Types, newDigest) {
		t.Errorf("Unmarshal() = %v, want %v", decoded.Types, newDigest)
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, tt := range allTests {
			dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			if err := Unmarshal(tt.want, &dst); (err != nil) != tt.wantErr {
				b.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		}
	}
}

func BenchmarkMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, tt := range allTests {
			// dst := reflect.New(reflect.TypeOf(tt.in)).Elem().Interface()
			if _, err := Marshal(tt.in); (err != nil) != tt.wantErr {
				b.Errorf("decodeState.unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		}
	}
}
