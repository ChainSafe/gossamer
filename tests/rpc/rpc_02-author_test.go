// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package rpc

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v2/scale"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/tests/utils"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v2"
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/stretchr/testify/require"
)

func TestAuthorSubmitExtrinsic(t *testing.T) {
	if utils.MODE != rpcSuite {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip RPC suite tests")
		return
	}

	t.Log("starting gossamer...")

	utils.CreateConfigBabeMaxThreshold()
	nodes, err := utils.InitializeAndStartNodes(t, 1, utils.GenesisDefault, utils.ConfigBABEMaxThreshold)
	require.NoError(t, err)

	defer func() {
		t.Log("going to tear down gossamer...")
		os.Remove(utils.ConfigBABEMaxThreshold)
		errList := utils.TearDown(t, nodes)
		require.Len(t, errList, 0)
	}()

	time.Sleep(15 * time.Second) // wait for server to start

	api, err := gsrpc.NewSubstrateAPI(fmt.Sprintf("http://localhost:%s", nodes[0].RPCPort))
	require.NoError(t, err)

	meta, err := api.RPC.State.GetMetadataLatest()
	require.NoError(t, err)

	// Create a call, transferring 12345 units to Bob
	bob, err := types.NewAddressFromHexAccountID("0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22")
	require.NoError(t, err)

	c, err := types.NewCall(meta, "Balances.transfer", bob, types.NewUCompactFromUInt(12345))
	require.NoError(t, err)

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	require.NoError(t, err)

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	require.NoError(t, err)

	key, err := types.CreateStorageKey(meta, "System", "Account", signature.TestKeyringPairAlice.PublicKey, nil)
	require.NoError(t, err)

	var accInfo types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accInfo)
	require.NoError(t, err)
	require.True(t, ok)

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		//Era:                types.ExtrinsicEra{IsImmortalEra: true},
		Era:                types.ExtrinsicEra{
			IsMortalEra:   true,
			AsMortalEra:   types.MortalEra{
				First:  132,
				Second: 1,
			},
		},
		GenesisHash:        genesisHash,
		//Nonce:              types.NewUCompactFromUInt(uint64(accInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	buffer := bytes.Buffer{}
	encoder := scale.NewEncoder(&buffer)
	ext.Encode(*encoder)
fmt.Printf("SUBMIT %x\n", buffer.Bytes())
	// Send the extrinsic
	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	require.NoError(t, err)
	require.NotEqual(t, hash, common.Hash{})
}

// TestDecodeExt is for debugging/decoding extrinsics.  Test with a hex string that was generated (from above tests
//  or polkadot.js/api) and use in buffer.Write.  The decoded output will show the values in the extrinsic.
func TestDecodeExt(t *testing.T) {
	buffer := bytes.Buffer{}
	decoder := scale.NewDecoder(&buffer)
	//buffer.Write(common.MustHexToBytes("0x410284ffd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01f8efbe48487e57a22abf7e3acd491b7f3528a33a111b1298601554863d27eb129eaa4e718e1365414ff3d028b62bebc651194c6b5001e5c2839b982757e08a8c0000000600ff8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a480b00c465f14670"))
	// todo failing from polkadot.js apps
	buffer.Write(common.MustHexToBytes("0x450284d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01380903d5f692eff4030ba9af1d8321e72cf828af54f64565766552b467c81914ecc82a10a00dbb70168c571b78fd61fd97f253f421c4091db4e7308c0cf1738e9604000006038eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48130010d2bd9f35b601"))
	// todo passing from test_transaction js
	//buffer.Write(common.MustHexToBytes("0x410284ffd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d015cbead600584b7701b8ab8384fd6550cda9de51525d861c2c7543f5ab323cf631ed69fceb4a93bccc576894f9b0d7ac8ae7b57c75e074b9c01238a72bd08f4830004000600ff8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48e5c0"))

	// todo from test_transaction (no options) fails
	//buffer.Write(common.MustHexToBytes("0x410284d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01b2d848cc5ce98c49afa180f7224ab38291e758e40a1d5bd1af9bc0d325c22c2f2d15a59b83791733355d4d4c5161e717334b2793ee6e0d739c175d241c70b98e8602040006038eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a480f0090c04bb6db2b"))
	ext := types.Extrinsic{}
	err := decoder.Decode(&ext)
	require.NoError(t, err)
	fmt.Printf("decoded ext %+v\n", ext)

}

func TestAuthorRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip RPC suite tests")
		return
	}

	testCases := []*testCase{
		{ //TODO
			description: "test author_submitExtrinsic",
			method:      "author_submitExtrinsic",
			skip:        true,
		},
		{ //TODO
			description: "test author_pendingExtrinsics",
			method:      "author_pendingExtrinsics",
			skip:        true,
		},
		{ //TODO
			description: "test author_removeExtrinsic",
			method:      "author_removeExtrinsic",
			skip:        true,
		},
		{ //TODO
			description: "test author_insertKey",
			method:      "author_insertKey",
			skip:        true,
		},
		{ //TODO
			description: "test author_rotateKeys",
			method:      "author_rotateKeys",
			skip:        true,
		},
		{ //TODO
			description: "test author_hasSessionKeys",
			method:      "author_hasSessionKeys",
			skip:        true,
		},
		{ //TODO
			description: "test author_hasKey",
			method:      "author_hasKey",
			skip:        true,
		},
	}

	t.Log("starting gossamer...")
	nodes, err := utils.InitializeAndStartNodes(t, 1, utils.GenesisDefault, utils.ConfigDefault)
	require.Nil(t, err)

	time.Sleep(time.Second) // give server a second to start

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			_ = getResponse(t, test)
		})
	}

	t.Log("going to tear down gossamer...")
	errList := utils.TearDown(t, nodes)
	require.Len(t, errList, 0)
}
