// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestEmptyHeader(t *testing.T) {
	head := NewEmptyHeader()
	isEmpty := head.Empty()
	require.True(t, isEmpty)

	head.Number = big.NewInt(21)
	isEmpty = head.Empty()
	require.False(t, isEmpty)

	vdts := NewDigest()
	err := vdts.Add(
		PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
	)
	require.NoError(t, err)

	head2, err := NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, big.NewInt(0), vdts)
	require.NoError(t, err)

	isEmpty = head2.Empty()
	require.False(t, isEmpty)

	head3, err := NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, big.NewInt(21), vdts)
	require.NoError(t, err)

	isEmpty = head3.Empty()
	require.False(t, isEmpty)
}

func TestEncodeAndDecodeHeader(t *testing.T) {
	expected, err := common.HexToBytes("0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d")
	require.NoError(t, err)

	vdts := NewDigest()
	err = vdts.Add(
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

	headerVdt, err := NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, big.NewInt(0), vdts)
	require.NoError(t, err)

	encVdt, err := scale.Marshal(*headerVdt)
	require.NoError(t, err)

	require.Equal(t, expected, encVdt)

	var decVdt = NewEmptyHeader()
	err = scale.Unmarshal(encVdt, decVdt)
	require.NoError(t, err)
	decVdt.Hash()
	require.Equal(t, headerVdt, decVdt)
}

func TestHeaderDeepCopy(t *testing.T) {
	vdts := NewDigest()
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

	header, err := NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, big.NewInt(1), vdts)
	require.NoError(t, err)

	dc, err := header.DeepCopy()
	require.NoError(t, err)
	dc.Hash()
	require.Equal(t, header, dc)
}
