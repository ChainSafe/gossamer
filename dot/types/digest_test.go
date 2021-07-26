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
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEncodeWithVdt(t *testing.T) {
	d := common.MustHexToBytes("0x0c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d")
	r := &bytes.Buffer{}
	_, _ = r.Write(d)
	//digest, err := DecodeDigest(r)
	//require.NoError(t, err)


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

	//type S struct {
	//	V scale.VaryingDataTypeSlice
	//}
	//
	//ts := S{V:vdts}
	bytes, err := scale.Marshal(vdts)
	require.NoError(t, err)
	require.Equal(t, d, bytes)

	v := DigestVdtSlice
	err = scale.Unmarshal(bytes, &v)
	//v, err := DecodeWithVdt(bytes)
	require.NoError(t, err)
	//fmt.Println(digest[0])
	//fmt.Println(v.Types[0].Value())
	//exp := digest[0]
	//act := v.Types[0].Value()
	//require.Equal(t, exp, act)


	//var act interface{}
	//switch val := v.Types[0].Value().(type) {
	//case ChangesTrieRootDigest:
	//	act = &val
	//case PreRuntimeDigest:
	//	act = &val
	//case ConsensusDigest:
	//	act = &val
	//case SealDigest:
	//	act = &val
	//}

	//require.Equal(t, exp, act)
	//fmt.Println(act)

	// Reencode and check
	encV, err := scale.Marshal(v)
	require.NoError(t, err)
	require.Equal(t, d, encV)

	//di, err := digest.Encode()
	//require.NoError(t, err)
	//require.Equal(t, d, di)
	//require.Equal(t, di, encV)
}

func TestDecodeSingleDigest(t *testing.T) {
	d := &PreRuntimeDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 3, 5, 7},
	}

	enc, err := d.Encode()
	require.NoError(t, err)

	r := &bytes.Buffer{}
	_, _ = r.Write(enc)
	decoder := scale.NewDecoder(r)
	d2, err := DecodeDigestItem(decoder)
	require.NoError(t, err)
	require.Equal(t, d2, d)
}

func TestDecodeDigestNew(t *testing.T) {
	d := common.MustHexToBytes("0x0c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d")
	r := &bytes.Buffer{}
	_, _ = r.Write(d)
	digest, err := DecodeDigest(r)
	require.NoError(t, err)
	require.Equal(t, 3, len(digest))

	enc, err := digest.Encode()
	require.NoError(t, err)
	require.Equal(t, d, enc)
}

func TestDecodeDigest(t *testing.T) {
	d := common.MustHexToBytes("0x0c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d")
	r := &bytes.Buffer{}
	_, _ = r.Write(d)
	digest := &Digest{}
	err := digest.Decode(r)
	require.NoError(t, err)
	require.Equal(t, 3, len(*digest))

	enc, err := digest.Encode()
	require.NoError(t, err)
	require.Equal(t, d, enc)
}

func TestChangesTrieRootDigest(t *testing.T) {
	d := &ChangesTrieRootDigest{
		Hash: common.Hash{0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152, 85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40, 246, 76, 38, 235, 204, 43, 31, 179, 28},
	}

	enc, err := d.Encode()
	require.NoError(t, err)

	r := &bytes.Buffer{}
	_, _ = r.Write(enc)
	dec := scale.NewDecoder(r)
	d2, err := DecodeDigestItem(dec)
	require.NoError(t, err)
	require.Equal(t, d, d2)
}

func TestPreRuntimeDigest(t *testing.T) {
	d := &PreRuntimeDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 3, 5, 7},
	}

	enc, err := d.Encode()
	require.NoError(t, err)

	r := &bytes.Buffer{}
	_, _ = r.Write(enc)
	dec := scale.NewDecoder(r)
	d2, err := DecodeDigestItem(dec)
	require.NoError(t, err)
	require.Equal(t, d, d2)
}

func TestConsensusDigest(t *testing.T) {
	d := &ConsensusDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 3, 5, 7},
	}

	enc, err := d.Encode()
	require.NoError(t, err)

	r := &bytes.Buffer{}
	_, _ = r.Write(enc)
	dec := scale.NewDecoder(r)
	d2, err := DecodeDigestItem(dec)
	require.NoError(t, err)
	require.Equal(t, d, d2)
}

func TestSealDigest(t *testing.T) {
	d := &SealDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 3, 5, 7},
	}

	enc, err := d.Encode()
	require.NoError(t, err)

	r := &bytes.Buffer{}
	_, _ = r.Write(enc)
	dec := scale.NewDecoder(r)
	d2, err := DecodeDigestItem(dec)
	require.NoError(t, err)
	require.Equal(t, d, d2)
}
