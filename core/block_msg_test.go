package core

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/keystore"
	"github.com/ChainSafe/gossamer/network"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/state"
	"github.com/ChainSafe/gossamer/tests"
	"github.com/ChainSafe/gossamer/trie"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/big"
	"testing"
	"time"
)

func TestProcessBlockRequestMsgType(t *testing.T) {
	testCases := []struct {
		name          string
		blockAnnounce *network.BlockAnnounceMessage
		msgType       int
		msgTypeString string
	}{
		//{
		//	name: "should respond with a BlockRequestMsgType",
		//	blockAnnounce: &network.BlockAnnounceMessage{
		//		Number:         big.NewInt(1),
		//		ParentHash:     common.Hash{},
		//		StateRoot:      common.Hash{},
		//		ExtrinsicsRoot: common.Hash{},
		//		Digest:         [][]byte{},
		//	},
		//	msgType:       network.BlockRequestMsgType, //1
		//	msgTypeString: "BlockRequestMsgType",
		//},
		{
			name: "should respond with a BlockAnnounceMessage",
			blockAnnounce: &network.BlockAnnounceMessage{
				Number:         big.NewInt(2),
				ParentHash:     common.Hash{},
				StateRoot:      common.Hash{},
				ExtrinsicsRoot: common.Hash{},
				Digest:         [][]byte{},
			},
			msgType:       network.BlockAnnounceMsgType, //3
			msgTypeString: "BlockAnnounceMsgType",
		},
	}

	rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

	for _, test := range testCases {

		localTest := test
		t.Run(test.name, func(t *testing.T) {

			msgSend := make(chan network.Message)
			newBlocks := make(chan types.Block)

			dataDir, err := ioutil.TempDir("", "./test_data")
			if err != nil {
				t.Fatal(err)
			}

			blockState := state.NewService(dataDir)

			err = blockState.Initialize(&types.Header{
				Number:    big.NewInt(0),
				StateRoot: trie.EmptyHash,
			}, trie.NewEmptyTrie(nil))
			if err != nil {
				t.Fatal(err)
			}

			err = blockState.Start()
			if err != nil {
				t.Fatal(err)
			}

			defer func() {
				err = blockState.Stop()
				if err != nil {
					t.Fatal(err)
				}
			}()

			// Create header
			header0 := &types.Header{
				Number: big.NewInt(0),
			}

			// BlockBody with fake extrinsics
			blockBody0 := types.Body{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

			block0 := &types.Block{
				Header: header0,
				Body:   &blockBody0,
			}

			// Add the block0 to the DB
			err = blockState.Block.AddBlock(block0)
			require.Nil(t, err)

			cfg := &Config{
				Runtime:    rt,
				MsgSend:    msgSend,
				Keystore:   keystore.NewKeystore(),
				BlockState: blockState.Block,
				NewBlocks:  newBlocks,
			}

			s, err := NewService(cfg)
			if err != nil {
				t.Fatal(err)
			}

			err = s.Start()
			if err != nil {
				t.Fatal(err)
			}
			defer s.Stop()

			if localTest.msgType == network.BlockAnnounceMsgType {
				// simulate block sent from BABE session
				newBlocks <- types.Block{
					Header: &types.Header{
						Number: big.NewInt(1),
					},
				}
			}

			select {
			case msg := <-msgSend:
				msgType := msg.GetType()
				require.Equal(t, localTest.msgType, msgType)
			case <-time.After(TestMessageTimeout):
				t.Error("timeout waiting for message")
			}
		})
	}
}
