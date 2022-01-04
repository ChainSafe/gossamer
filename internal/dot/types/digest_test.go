// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	d := common.MustHexToBytes("0x0c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d") //nolint:lll
	r := &bytes.Buffer{}
	_, _ = r.Write(d)

	vdts := NewDigest()
	err := vdts.Add(
		PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
		ConsensusDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"), //nolint:lll
		},
		SealDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"), //nolint:lll
		},
	)
	require.NoError(t, err)

	b, err := scale.Marshal(vdts)
	require.NoError(t, err)
	require.Equal(t, d, b)

	v := NewDigest()
	err = scale.Unmarshal(b, &v)
	require.NoError(t, err)

	encV, err := scale.Marshal(v)
	require.NoError(t, err)
	require.Equal(t, d, encV)
}

func TestDecodeSingleDigest(t *testing.T) {
	exp := common.MustHexToBytes("0x06424142451001030507")
	d := PreRuntimeDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 3, 5, 7},
	}

	di := NewDigestItem()
	err := di.Set(d)
	require.NoError(t, err)

	enc, err := scale.Marshal(di)
	require.NoError(t, err)

	require.Equal(t, exp, enc)

	v := NewDigestItem()
	err = scale.Unmarshal(enc, &v)
	require.NoError(t, err)

	require.Equal(t, di.Value(), v.Value())
}

func TestDecodeDigest(t *testing.T) {
	d := common.MustHexToBytes("0x0c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d") //nolint:lll

	v := NewDigest()
	err := scale.Unmarshal(d, &v)
	require.NoError(t, err)
	require.Equal(t, 3, len(v.Types))

	enc, err := scale.Marshal(v)
	require.NoError(t, err)
	require.Equal(t, d, enc)
}

func TestChangesTrieRootDigest(t *testing.T) {
	exp := common.MustHexToBytes("0x02005b3219d65e772447d8219855b822783da1a4df4c3528f64c26ebcc2b1fb31c")
	d := ChangesTrieRootDigest{
		Hash: common.Hash{
			0, 91, 50, 25, 214, 94, 119, 36, 71,
			216, 33, 152, 85, 184, 34, 120, 61,
			161, 164, 223, 76, 53, 40, 246, 76,
			38, 235, 204, 43, 31, 179, 28},
	}

	di := NewDigestItem()
	err := di.Set(d)
	require.NoError(t, err)

	enc, err := scale.Marshal(di)
	require.NoError(t, err)

	require.Equal(t, exp, enc)

	v := NewDigestItem()
	err = scale.Unmarshal(enc, &v)
	require.NoError(t, err)

	require.Equal(t, di.Value(), v.Value())
}

func TestPreRuntimeDigest(t *testing.T) {
	exp := common.MustHexToBytes("0x06424142451001030507")
	d := PreRuntimeDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 3, 5, 7},
	}

	di := NewDigestItem()
	err := di.Set(d)
	require.NoError(t, err)

	enc, err := scale.Marshal(di)
	require.NoError(t, err)

	require.Equal(t, exp, enc)

	v := NewDigestItem()
	err = scale.Unmarshal(enc, &v)
	require.NoError(t, err)

	require.Equal(t, di.Value(), v.Value())
}

func TestConsensusDigest(t *testing.T) {
	exp := common.MustHexToBytes("0x04424142451001030507")
	d := ConsensusDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 3, 5, 7},
	}

	di := NewDigestItem()
	err := di.Set(d)
	require.NoError(t, err)

	enc, err := scale.Marshal(di)
	require.NoError(t, err)

	require.Equal(t, exp, enc)

	v := NewDigestItem()
	err = scale.Unmarshal(enc, &v)
	require.NoError(t, err)

	require.Equal(t, di.Value(), v.Value())
}

func TestSealDigest(t *testing.T) {
	exp := common.MustHexToBytes("0x05424142451001030507")
	d := SealDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 3, 5, 7},
	}

	di := NewDigestItem()
	err := di.Set(d)
	require.NoError(t, err)

	enc, err := scale.Marshal(di)
	require.NoError(t, err)

	require.Equal(t, exp, enc)

	v := NewDigestItem()
	err = scale.Unmarshal(enc, &v)
	require.NoError(t, err)

	require.Equal(t, di.Value(), v.Value())
}
