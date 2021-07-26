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

package types

import (
	"bytes"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/stretchr/testify/require"
)

type MyVaryingDataTypeSliceStruct struct {
	VDTS scale.VaryingDataTypeSlice
}

func TestOptionals(t *testing.T) {
	f := optional.NewHeaderVdt(false, nil)
	tr := optional.NewHeaderVdt(true, nil)

	enc, err := scale.Marshal(f)
	require.NoError(t, err)

	enc2, err := scale.Marshal(tr)
	require.NoError(t, err)

	var opt1 optional.HeaderVdt
	err = scale.Unmarshal(enc, &opt1)
	require.NoError(t, err)

	var opt2 optional.HeaderVdt
	err = scale.Unmarshal(enc2, &opt2)
	require.NoError(t, err)

	require.Equal(t, opt1, f)
	require.Equal(t, opt2, tr)
	// All works
}

func TestSomething(t *testing.T) {
	var dVdt = scale.MustNewVaryingDataType(ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{})
	var vdts = scale.NewVaryingDataTypeSlice(dVdt)

	err := vdts.Add(
		PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
		ConsensusDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		SealDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
		},
	)
	require.NoError(t, err)

	encVdt, err := scale.Marshal(MyVaryingDataTypeSliceStruct{vdts})
	require.NoError(t, err)

	var vdts1 = scale.NewVaryingDataTypeSlice(dVdt)
	mvdtss := MyVaryingDataTypeSliceStruct{
		VDTS: vdts1,
	}
	err = scale.Unmarshal(encVdt, &mvdtss)
	require.NoError(t, err)

	require.Equal(t, MyVaryingDataTypeSliceStruct{vdts}, mvdtss)
}

func TestEncodeAndDecodeHeaderVdt(t *testing.T) {
	vdts := DigestVdtSlice
	err := vdts.Add(
		PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
		ConsensusDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		SealDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
		},
	)
	require.NoError(t, err)

	d := Digest{
		&PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
		&ConsensusDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		&SealDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
		},
	}

	header, err := NewHeader(common.Hash{1}, common.Hash{1}, common.Hash{1}, big.NewInt(1), d)
	require.NoError(t, err)

	headerVdt, err := NewHeaderVdt(common.Hash{1}, common.Hash{1}, common.Hash{1}, big.NewInt(1), vdts)
	require.NoError(t, err)

	require.Equal(t, header.hash, headerVdt.hash)

	enc, err := header.Encode()
	require.NoError(t, err)

	encVdt, err := scale.Marshal(*headerVdt)
	require.NoError(t, err)

	require.Equal(t, enc, encVdt)

	rw := &bytes.Buffer{}
	rw.Write(enc)
	dec, err := new(Header).Decode(rw)
	require.NoError(t, err)

	var decVdt = NewEmptyHeaderVdt()
	err = scale.Unmarshal(encVdt, decVdt)
	require.NoError(t, err)

	 // Test if reencoding is equal
	enc2, err := dec.Encode()
	require.NoError(t, err)

	encVdt2, err := scale.Marshal(*decVdt)
	require.NoError(t, err)

	require.Equal(t, enc2, encVdt2)
	// Should test decoding optional but honestly lets just get rid of it
}

func TestOptionalEncoding(t *testing.T) {
	vdts := DigestVdtSlice
	header, err := NewHeader(common.Hash{1}, common.Hash{1}, common.Hash{1}, big.NewInt(1), Digest{})
	require.NoError(t, err)

	headerVdt, err := NewHeaderVdt(common.Hash{1}, common.Hash{1}, common.Hash{1}, big.NewInt(1), vdts)
	require.NoError(t, err)

	//option := header.AsOptional()

	vdtOption := headerVdt.AsOptional()

	enc, err := scale.Marshal(*header)
	require.NoError(t, err)

	encVdt, err := scale.Marshal(*vdtOption.Value)
	require.NoError(t, err)

	encVdtOption, err := scale.Marshal(*vdtOption)
	require.NoError(t, err)

	require.Equal(t, enc, encVdt)

	var dec = optional.HeaderVdt{Value: &optional.CoreHeaderVdt{Digest: vdts}}
	err = scale.Unmarshal(encVdtOption, &dec)
	require.NoError(t, err)
	require.Equal(t, *vdtOption, dec)
	// Works to this point
}

//func TestOptionalDecoding(t *testing.T) {
//	vdts := DigestVdtSlice
//	err := vdts.Add(
//		PreRuntimeDigest{
//			ConsensusEngineID: BabeEngineID,
//			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
//		},
//		ConsensusDigest{
//			ConsensusEngineID: BabeEngineID,
//			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
//		},
//		SealDigest{
//			ConsensusEngineID: BabeEngineID,
//			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
//		},
//	)
//	require.NoError(t, err)
//
//	d := Digest{
//		&PreRuntimeDigest{
//			ConsensusEngineID: BabeEngineID,
//			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
//		},
//		&ConsensusDigest{
//			ConsensusEngineID: BabeEngineID,
//			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
//		},
//		&SealDigest{
//			ConsensusEngineID: BabeEngineID,
//			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
//		},
//	}
//
//	header, err := NewHeader(common.Hash{1}, common.Hash{1}, common.Hash{1}, big.NewInt(1), d)
//	require.NoError(t, err)
//
//	headerVdt, err := NewHeaderVdt(common.Hash{1}, common.Hash{1}, common.Hash{1}, big.NewInt(1), vdts)
//	require.NoError(t, err)
//
//	//headerVdtPointer := &headerVdt
//	//fmt.Println(headerVdtPointer)
//	optional := header.AsOptional()
//	fmt.Println(optional.String())
//
//	enc, err := scale.Marshal(optional)
//	require.NoError(t, err)
//
//	//encVdt, err := scale.Marshal(headerVdtPointer)
//	//require.NoError(t, err)
//
//	rw := &bytes.Buffer{}
//	rw.Write(enc)
//
//	opt, err := decodeOptionalHeader(rw)
//	require.NoError(t, err)
//	require.Equal(t, optional, opt)
//
//
//}

// TODO add test for deep copy of VDTs

func TestHeaderVdtAsOptional(t *testing.T) {
	vdts := DigestVdtSlice
	err := vdts.Add(
		PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
		ConsensusDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		SealDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
		},
	)
	require.NoError(t, err)

	headerVdt, err := NewHeaderVdt(common.Hash{1}, common.Hash{1}, common.Hash{1}, big.NewInt(1), vdts)
	require.NoError(t, err)

	optionalVdt := headerVdt.AsOptional()
	require.True(t, optionalVdt.Exists)
	opHead := optionalVdt.Value
	require.NotNil(t, opHead)
	require.Equal(t, headerVdt.Digest, opHead.Digest)

	//TODO test other functions or just get rid of this optional type cuz i dont think its necessary with new scale

	enc, err := scale.Marshal(*optionalVdt)
	require.NoError(t, err)

	//Decode
	opt, err := decodeOptionalVdtHeader(enc)
	require.NoError(t, err)
	require.Equal(t, optionalVdt, opt)

	//newHead, err := NewHeaderVdtFromOptional(optionalVdt)
	//require.NoError(t, err)
	//require.Equal(t, headerVdt, newHead)
}

func TestDecodeHeader(t *testing.T) {
	header, err := NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, big.NewInt(0), Digest{})
	require.NoError(t, err)

	enc, err := header.Encode()
	require.NoError(t, err)

	rw := &bytes.Buffer{}
	rw.Write(enc)
	dec, err := new(Header).Decode(rw)
	require.NoError(t, err)
	dec.Hash()
	require.Equal(t, header, dec)
}

func TestMustEncodeHeader(t *testing.T) {
	bh1, err := NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, big.NewInt(0), Digest{})
	require.NoError(t, err)
	enc, err := bh1.Encode()
	require.NoError(t, err)

	testDigest := Digest{
		&PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              []byte{1, 2, 3},
		},
		&SealDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              []byte{4, 5, 6, 7},
		},
	}

	bh2, err := NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, big.NewInt(0), testDigest)
	require.NoError(t, err)
	enc2, err := bh2.Encode()
	require.NoError(t, err)

	tests := []struct {
		name string
		take *Header
		want []byte
	}{
		{
			name: "correct",
			take: bh1,
			want: enc,
		},
		{
			name: "correct2",
			take: bh2,
			want: enc2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.take.MustEncode(); !bytes.Equal(got, tt.want) {
				t.Errorf("MustEncode() = %v, want %v", got, tt.want)
			}
		})
	}
}