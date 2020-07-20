package sync

import (
	"testing"
)

// BlockRequestMessage 1

// tests the ProcessBlockRequestMessage method
func TestService_ProcessBlockRequestMessage(t *testing.T) {
	msgSend := make(chan network.Message, 10)

	cfg := &Config{
		MsgSend: msgSend,
	}

	s := NewTestService(t, cfg)
	s.started.Store(true)

	addTestBlocksToState(t, 2, s.blockState)

	bestHash := s.blockState.BestBlockHash()
	bestBlock, err := s.blockState.GetBlockByNumber(big.NewInt(1))
	require.Nil(t, err)

	// set some nils and check no error is thrown
	bds := &types.BlockData{
		Hash:          bestHash,
		Header:        nil,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}
	err = s.blockState.CompareAndSetBlockData(bds)
	require.Nil(t, err)

	// set receipt message and justification
	bds = &types.BlockData{
		Hash:          bestHash,
		Receipt:       optional.NewBytes(true, []byte("asdf")),
		MessageQueue:  optional.NewBytes(true, []byte("ghjkl")),
		Justification: optional.NewBytes(true, []byte("qwerty")),
	}

	endHash := s.blockState.BestBlockHash()
	start, err := variadic.NewUint64OrHash(uint64(1))
	if err != nil {
		t.Fatal(err)
	}

	err = s.blockState.CompareAndSetBlockData(bds)

	require.Nil(t, err)

	testsCases := []struct {
		description      string
		value            *network.BlockRequestMessage
		expectedMsgType  int
		expectedMsgValue *network.BlockResponseMessage
	}{
		{
			description: "test get Header and Body",
			value: &network.BlockRequestMessage{
				ID:            1,
				RequestedData: 3,
				StartingBlock: start,
				EndBlockHash:  optional.NewHash(true, endHash),
				Direction:     1,
				Max:           optional.NewUint32(false, 0),
			},
			expectedMsgType: network.BlockResponseMsgType,
			expectedMsgValue: &network.BlockResponseMessage{
				ID: 1,
				BlockData: []*types.BlockData{
					{
						Hash:   optional.NewHash(true, bestHash).Value(),
						Header: bestBlock.Header.AsOptional(),
						Body:   bestBlock.Body.AsOptional(),
					},
				},
			},
		},
		{
			description: "test get Header",
			value: &network.BlockRequestMessage{
				ID:            2,
				RequestedData: 1,
				StartingBlock: start,
				EndBlockHash:  optional.NewHash(true, endHash),
				Direction:     1,
				Max:           optional.NewUint32(false, 0),
			},
			expectedMsgType: network.BlockResponseMsgType,
			expectedMsgValue: &network.BlockResponseMessage{
				ID: 2,
				BlockData: []*types.BlockData{
					{
						Hash:   optional.NewHash(true, bestHash).Value(),
						Header: bestBlock.Header.AsOptional(),
						Body:   optional.NewBody(false, nil),
					},
				},
			},
		},
		{
			description: "test get Receipt",
			value: &network.BlockRequestMessage{
				ID:            2,
				RequestedData: 4,
				StartingBlock: start,
				EndBlockHash:  optional.NewHash(true, endHash),
				Direction:     1,
				Max:           optional.NewUint32(false, 0),
			},
			expectedMsgType: network.BlockResponseMsgType,
			expectedMsgValue: &network.BlockResponseMessage{
				ID: 2,
				BlockData: []*types.BlockData{
					{
						Hash:    optional.NewHash(true, bestHash).Value(),
						Header:  optional.NewHeader(false, nil),
						Body:    optional.NewBody(false, nil),
						Receipt: bds.Receipt,
					},
				},
			},
		},
		{
			description: "test get MessageQueue",
			value: &network.BlockRequestMessage{
				ID:            2,
				RequestedData: 8,
				StartingBlock: start,
				EndBlockHash:  optional.NewHash(true, endHash),
				Direction:     1,
				Max:           optional.NewUint32(false, 0),
			},
			expectedMsgType: network.BlockResponseMsgType,
			expectedMsgValue: &network.BlockResponseMessage{
				ID: 2,
				BlockData: []*types.BlockData{
					{
						Hash:         optional.NewHash(true, bestHash).Value(),
						Header:       optional.NewHeader(false, nil),
						Body:         optional.NewBody(false, nil),
						MessageQueue: bds.MessageQueue,
					},
				},
			},
		},
		{
			description: "test get Justification",
			value: &network.BlockRequestMessage{
				ID:            2,
				RequestedData: 16,
				StartingBlock: start,
				EndBlockHash:  optional.NewHash(true, endHash),
				Direction:     1,
				Max:           optional.NewUint32(false, 0),
			},
			expectedMsgType: network.BlockResponseMsgType,
			expectedMsgValue: &network.BlockResponseMessage{
				ID: 2,
				BlockData: []*types.BlockData{
					{
						Hash:          optional.NewHash(true, bestHash).Value(),
						Header:        optional.NewHeader(false, nil),
						Body:          optional.NewBody(false, nil),
						Justification: bds.Justification,
					},
				},
			},
		},
	}

	for _, test := range testsCases {
		t.Run(test.description, func(t *testing.T) {

			err := s.ProcessBlockRequestMessage(test.value)
			require.Nil(t, err)

			select {
			case resp := <-msgSend:
				msgType := resp.GetType()

				require.Equal(t, test.expectedMsgType, msgType)

				require.Equal(t, test.expectedMsgValue.ID, resp.(*network.BlockResponseMessage).ID)

				require.Len(t, resp.(*network.BlockResponseMessage).BlockData, 2)

				require.Equal(t, test.expectedMsgValue.BlockData[0].Hash, bestHash)
				require.Equal(t, test.expectedMsgValue.BlockData[0].Header, resp.(*network.BlockResponseMessage).BlockData[0].Header)
				require.Equal(t, test.expectedMsgValue.BlockData[0].Body, resp.(*network.BlockResponseMessage).BlockData[0].Body)

				if test.expectedMsgValue.BlockData[0].Receipt != nil {
					require.Equal(t, test.expectedMsgValue.BlockData[0].Receipt, resp.(*network.BlockResponseMessage).BlockData[1].Receipt)
				}

				if test.expectedMsgValue.BlockData[0].MessageQueue != nil {
					require.Equal(t, test.expectedMsgValue.BlockData[0].MessageQueue, resp.(*network.BlockResponseMessage).BlockData[1].MessageQueue)
				}

				if test.expectedMsgValue.BlockData[0].Justification != nil {
					require.Equal(t, test.expectedMsgValue.BlockData[0].Justification, resp.(*network.BlockResponseMessage).BlockData[1].Justification)
				}
			case <-time.After(testMessageTimeout):
				t.Error("timeout waiting for message")
			}
		})
	}
}

// BlockResponseMessage 2

// tests the ProcessBlockResponseMessage method
func TestService_ProcessBlockResponseMessage(t *testing.T) {
	tt := trie.NewEmptyTrie()
	rt := runtime.NewTestRuntimeWithTrie(t, runtime.NODE_RUNTIME, tt, log.LvlTrace)

	kp, err := sr25519.GenerateKeypair()
	require.Nil(t, err)

	ks := keystore.NewKeystore()
	ks.Insert(kp)
	msgSend := make(chan network.Message, 10)

	cfg := &Config{
		Runtime:         rt,
		Keystore:        ks,
		IsBlockProducer: false,
		MsgSend:         msgSend,
	}

	s := NewTestService(t, cfg)

	hash := common.NewHash([]byte{0})
	body := optional.CoreBody{0xa, 0xb, 0xc, 0xd}

	parentHash := testGenesisHeader.Hash()
	stateRoot, err := common.HexToHash("0x2747ab7c0dc38b7f2afba82bd5e2d6acef8c31e09800f660b75ec84a7005099f")
	require.Nil(t, err)

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	require.Nil(t, err)

	preDigest, err := common.HexToBytes("0x014241424538e93dcef2efc275b72b4fa748332dc4c9f13be1125909cf90c8e9109c45da16b04bc5fdf9fe06a4f35e4ae4ed7e251ff9ee3d0d840c8237c9fb9057442dbf00f210d697a7b4959f792a81b948ff88937e30bf9709a8ab1314f71284da89a40000000000000000001100000000000000")
	if err != nil {
		t.Fatal(err)
	}

	header := &types.Header{
		ParentHash:     parentHash,
		Number:         big.NewInt(1),
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         [][]byte{preDigest},
	}

	bds := []*types.BlockData{{
		Hash:          header.Hash(),
		Header:        header.AsOptional(),
		Body:          types.NewBody([]byte{}).AsOptional(),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}, {
		Hash:          hash,
		Header:        optional.NewHeader(false, nil),
		Body:          optional.NewBody(true, body),
		Receipt:       optional.NewBytes(true, []byte("asdf")),
		MessageQueue:  optional.NewBytes(true, []byte("ghjkl")),
		Justification: optional.NewBytes(true, []byte("qwerty")),
	}}

	blockResponse := &network.BlockResponseMessage{
		BlockData: bds,
	}

	err = s.blockState.SetHeader(header)
	require.Nil(t, err)

	err = s.ProcessBlockResponseMessage(blockResponse)
	require.Nil(t, err)

	select {
	case resp := <-s.syncer.respIn:
		msgType := resp.GetType()
		require.Equal(t, network.BlockResponseMsgType, msgType)
	case <-time.After(testMessageTimeout):
		t.Error("timeout waiting for message")
	}
}
