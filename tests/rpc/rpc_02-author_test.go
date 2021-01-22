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

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/tests/utils"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v2"
	"github.com/centrifuge/go-substrate-rpc-client/v2/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v2/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/stretchr/testify/require"
)

func TestAuthorSubmitExtrinsic(t *testing.T) {
	if utils.MODE != rpcSuite {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip RPC suite tests")
		//return
	}

	t.Log("starting gossamer...")

	utils.CreateConfigBabeMaxThreshold()

	nodes, err := utils.InitializeAndStartNodes(t, 1, utils.GenesisDefault, utils.ConfigDefault)
	require.NoError(t, err)

	defer func() {
		t.Log("going to tear down gossamer...")
		os.Remove(utils.ConfigDefault)
		errList := utils.TearDown(t, nodes)
		require.Len(t, errList, 0)
	}()

	time.Sleep(10 * time.Second) // wait for server to start

	api, err := gsrpc.NewSubstrateAPI(fmt.Sprintf("http://localhost:%s", nodes[0].RPCPort))
	require.NoError(t, err)

	meta, err := api.RPC.State.GetMetadataLatest()
	require.NoError(t, err)

	// Create a call, transferring 12345 units to Bob
	bob, err := types.NewAddressFromHexAccountID("0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22")
	require.NoError(t, err)

	c, err := types.NewCall(meta, "Balances.transfer", bob, types.NewUCompactFromUInt(12345))
	if err != nil {
		panic(err)
	}

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	// TODO: the genesis hash is wrong.
	// genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	// require.NoError(t, err)

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	require.NoError(t, err)

	key, err := types.CreateStorageKey(meta, "System", "Account", signature.TestKeyringPairAlice.PublicKey, nil)
	require.NoError(t, err)

	var nonce uint32
	ok, err := api.RPC.State.GetStorageLatest(key, &nonce)
	require.NoError(t, err)
	require.True(t, ok)

	// this is the actual genesis hash for GenesisDefault
	genesisHash := types.NewHash(common.MustHexToBytes("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314"))

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsImmortalEra: true},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: 1, //rv.TransactionVersion, // TODO: rv.TransactionVersion == 0 but runtime expects 1
	}

	// // Sign the transaction using Alice's default account
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	w := &bytes.Buffer{}
	encoder := scale.NewEncoder(w)
	err = ext.Encode(*encoder)
	require.NoError(t, err)
	enc := w.Bytes()
	t.Logf("extrinsic 0x%x", enc)

	enc, err = types.EncodeToBytes(ext.Method)
	require.NoError(t, err)
	t.Logf("Method 0x%x", enc)

	enc, err = types.EncodeToBytes(o.Era)
	require.NoError(t, err)
	t.Logf("Era 0x%x", enc)

	enc, err = types.EncodeToBytes(o.Nonce)
	require.NoError(t, err)
	t.Logf("Nonce 0x%x", enc)

	enc, err = types.EncodeToBytes(o.Tip)
	require.NoError(t, err)
	t.Logf("Tip 0x%x", enc)

	enc, err = types.EncodeToBytes(o.SpecVersion)
	require.NoError(t, err)
	t.Logf("SpecVersion 0x%x", enc)

	enc, err = types.EncodeToBytes(o.TransactionVersion)
	require.NoError(t, err)
	t.Logf("TransactionVersion 0x%x", enc)

	t.Logf("genesisHash 0x%x", genesisHash)

	// Send the extrinsic
	hash, err := api.RPC.Author.SubmitExtrinsic(ext)
	require.NoError(t, err)

	fmt.Printf("Transfer sent with hash %#x\n", hash)
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
