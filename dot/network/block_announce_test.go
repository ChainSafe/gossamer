// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"math/big"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestEncodeBlockAnnounce(t *testing.T) {
	expected := common.MustHexToBytes("0x01000000000000000000000000000000000000000000000000000000000000003501020000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d00")

	digestVdt := types.NewDigest()
	err := digestVdt.Add(
		types.PreRuntimeDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
		types.ConsensusDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		types.SealDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
		},
	)
	require.NoError(t, err)

	testBlockAnnounce := BlockAnnounceMessage{
		ParentHash:     common.Hash{1},
		Number:         big.NewInt(77),
		StateRoot:      common.Hash{2},
		ExtrinsicsRoot: common.Hash{3},
		Digest:         digestVdt,
	}

	enc, err := scale.Marshal(testBlockAnnounce)
	require.NoError(t, err)

	require.Equal(t, expected, enc)
}

func TestDecodeBlockAnnounce(t *testing.T) {
	enc := common.MustHexToBytes("0x01000000000000000000000000000000000000000000000000000000000000003501020000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d00")

	digestVdt := types.NewDigest()
	err := digestVdt.Add(
		types.PreRuntimeDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
		types.ConsensusDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		types.SealDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
		},
	)
	require.NoError(t, err)

	expected := BlockAnnounceMessage{
		ParentHash:     common.Hash{1},
		Number:         big.NewInt(77),
		StateRoot:      common.Hash{2},
		ExtrinsicsRoot: common.Hash{3},
		Digest:         digestVdt,
	}

	act := BlockAnnounceMessage{
		Number: big.NewInt(0),
		Digest: types.NewDigest(),
	}
	err = scale.Unmarshal(enc, &act)
	require.NoError(t, err)

	require.Equal(t, expected, act)
}

func TestEncodeBlockAnnounceHandshake(t *testing.T) {
	expected := common.MustHexToBytes("0x044d00000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000")
	testHandshake := BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}

	enc, err := scale.Marshal(testHandshake)
	require.NoError(t, err)
	require.Equal(t, expected, enc)
}

func TestDecodeBlockAnnounceHandshake(t *testing.T) {
	enc := common.MustHexToBytes("0x044d00000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000")
	expected := BlockAnnounceHandshake{
		Roles:           4,
		BestBlockNumber: 77,
		BestBlockHash:   common.Hash{1},
		GenesisHash:     common.Hash{2},
	}

	msg := BlockAnnounceHandshake{}
	err := scale.Unmarshal(enc, &msg)
	require.NoError(t, err)
	require.Equal(t, expected, msg)
}

func TestHandleBlockAnnounceMessage(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	s := createTestService(t, config)

	peerID := peer.ID("noot")
	msg := &BlockAnnounceMessage{
		Number: big.NewInt(10),
		Digest: types.NewDigest(),
	}

	propagate, err := s.handleBlockAnnounceMessage(peerID, msg)
	require.NoError(t, err)
	require.True(t, propagate)
}

func TestValidateBlockAnnounceHandshake(t *testing.T) {
	configA := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true
	nodeA.notificationsProtocols[BlockAnnounceMsgType] = &notificationsProtocol{
		inboundHandshakeData: new(sync.Map),
	}
	testPeerID := peer.ID("noot")
	nodeA.notificationsProtocols[BlockAnnounceMsgType].inboundHandshakeData.Store(testPeerID, handshakeData{})

	err := nodeA.validateBlockAnnounceHandshake(testPeerID, &BlockAnnounceHandshake{
		BestBlockNumber: 100,
		GenesisHash:     nodeA.blockState.GenesisHash(),
	})
	require.NoError(t, err)
}
