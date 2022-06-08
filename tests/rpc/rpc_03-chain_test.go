// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	libutils "github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/retry"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	regex32BytesHex = `^0x[0-9a-f]{64}$`
	regexBytesHex   = `^0x[0-9a-f]{2}[0-9a-f]*$`
)

func TestChainRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	tomlConfig.Core.BABELead = true
	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	// Wait for Gossamer to produce block 2
	errBlockNumberTooHigh := errors.New("block number is too high")
	const retryWaitDuration = 200 * time.Millisecond
	err := retry.UntilOK(ctx, retryWaitDuration, func() (ok bool, err error) {
		var header modules.ChainBlockHeaderResponse
		fetchWithTimeout(ctx, t, "chain_getHeader", "[]", &header)
		number, err := common.HexToUint(header.Number)
		if err != nil {
			return false, fmt.Errorf("cannot convert header number to uint: %w", err)
		}

		switch number {
		case 0, 1:
			return false, nil
		case 2:
			return true, nil
		default:
			return false, fmt.Errorf("%w: %d", errBlockNumberTooHigh, number)
		}
	})
	require.NoError(t, err)

	var finalizedHead string
	fetchWithTimeout(ctx, t, "chain_getFinalizedHead", "[]", &finalizedHead)
	assert.Regexp(t, regex32BytesHex, finalizedHead)

	var header modules.ChainBlockHeaderResponse
	fetchWithTimeout(ctx, t, "chain_getHeader", "[]", &header)

	// Check and clear unpredictable fields
	assert.Regexp(t, regex32BytesHex, header.StateRoot)
	header.StateRoot = ""
	assert.Regexp(t, regex32BytesHex, header.ExtrinsicsRoot)
	header.ExtrinsicsRoot = ""
	assert.Len(t, header.Digest.Logs, 2)
	for _, digestLog := range header.Digest.Logs {
		assert.Regexp(t, regexBytesHex, digestLog)
	}
	header.Digest.Logs = nil

	// Assert remaining struct with predictable fields
	expectedHeader := modules.ChainBlockHeaderResponse{
		ParentHash: finalizedHead,
		Number:     "0x02",
	}
	assert.Equal(t, expectedHeader, header)

	var block modules.ChainBlockResponse
	fetchWithTimeout(ctx, t, "chain_getBlock", fmt.Sprintf(`["`+header.ParentHash+`"]`), &block)

	// Check and clear unpredictable fields
	assert.Regexp(t, regex32BytesHex, block.Block.Header.ParentHash)
	block.Block.Header.ParentHash = ""
	assert.Regexp(t, regex32BytesHex, block.Block.Header.StateRoot)
	block.Block.Header.StateRoot = ""
	assert.Regexp(t, regex32BytesHex, block.Block.Header.ExtrinsicsRoot)
	block.Block.Header.ExtrinsicsRoot = ""
	assert.Len(t, block.Block.Header.Digest.Logs, 3)
	for _, digestLog := range block.Block.Header.Digest.Logs {
		assert.Regexp(t, regexBytesHex, digestLog)
	}
	block.Block.Header.Digest.Logs = nil
	assert.Len(t, block.Block.Body, 1)
	const bodyRegex = `^0x280403000b[0-9a-z]{8}8101$`
	assert.Regexp(t, bodyRegex, block.Block.Body[0])
	block.Block.Body = nil

	// Assert remaining struct with predictable fields
	expectedBlock := modules.ChainBlockResponse{
		Block: modules.ChainBlock{
			Header: modules.ChainBlockHeaderResponse{
				Number: "0x01",
			},
		},
	}
	assert.Equal(t, expectedBlock, block)

	var blockHash string
	fetchWithTimeout(ctx, t, "chain_getBlockHash", "[]", &blockHash)
	assert.Regexp(t, regex32BytesHex, blockHash)
	assert.NotEqual(t, finalizedHead, blockHash)
}

func TestChainSubscriptionRPC(t *testing.T) {
	if utils.MODE != rpcSuite {
		t.Log("Going to skip RPC suite tests")
		return
	}

	testCases := []*testCase{
		{
			description: "test chain_subscribeNewHeads",
			method:      "chain_subscribeNewHeads",
			expected: []interface{}{1,
				map[string](interface{}){
					"subscription": float64(1),
					"result": map[string](interface{}){
						"number":         "0x01",
						"parentHash":     "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21",
						"stateRoot":      "0x3b1a31d10d4d8a444579fd5a3fb17cbe6bebba9d939d88fe7bafb9d48036abb5",
						"extrinsicsRoot": "0x8025c0d64df303f79647611c8c2b0a77bc2247ee12d851df4624e1f71ebb3aed",
						//nolint:lll
						"digest": map[string](interface{}){"logs": []interface{}{
							"0x0642414245c101c809062df1d1271d6a50232754baa64870515a7ada927886467748a220972c6d58347fd7317e286045604c5ddb78b84018c4b3a3836ee6626c8da6957338720053588d9f29c307fade658661d8d6a57c525f48553a253cf6e1475dbd319ca90200000000000000000e00000000000000",
							"0x054241424501017cac567e5b5688260d9d0a1f7fe6a9f81ae0f1900a382e1c73a4929fcaf6e33ed9e7347eb81ebb2699d58f6c8b01c7bdf0714e5f6f4495bc4b5fb3becb287580"}}}}},
			params: "[]",
			skip:   false,
		},
		{
			description: "test state_subscribeStorage",
			method:      "state_subscribeStorage",
			expected:    "",
			params:      "[]",
			skip:        true,
		},
		{
			description: "test chain_finalizedHeads",
			method:      "chain_subscribeFinalizedHeads",
			expected: []interface{}{1,
				map[string](interface{}){
					"subscription": float64(1),
					"result": map[string](interface{}){
						"number":         "0x01",
						"parentHash":     "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21",
						"stateRoot":      "0x3b1a31d10d4d8a444579fd5a3fb17cbe6bebba9d939d88fe7bafb9d48036abb5",
						"extrinsicsRoot": "0x8025c0d64df303f79647611c8c2b0a77bc2247ee12d851df4624e1f71ebb3aed",
						//nolint:lll
						"digest": map[string](interface{}){"logs": []interface{}{
							"0x0642414245c101c809062df1d1271d6a50232754baa64870515a7ada927886467748a220972c6d58347fd7317e286045604c5ddb78b84018c4b3a3836ee6626c8da6957338720053588d9f29c307fade658661d8d6a57c525f48553a253cf6e1475dbd319ca90200000000000000000e00000000000000",
							"0x054241424501017cac567e5b5688260d9d0a1f7fe6a9f81ae0f1900a382e1c73a4929fcaf6e33ed9e7347eb81ebb2699d58f6c8b01c7bdf0714e5f6f4495bc4b5fb3becb287580"}}}}},
			params: "[]",
			skip:   false,
		},
	}

	genesisPath := libutils.GetDevGenesisSpecPathTest(t)
	tomlConfig := config.Default()
	tomlConfig.Init.Genesis = genesisPath
	tomlConfig.Core.BABELead = true
	tomlConfig.RPC.WS = true // WS port is set in the node.New constructor
	node := node.New(t, tomlConfig)
	ctx, cancel := context.WithCancel(context.Background())
	node.InitAndStartTest(ctx, t, cancel)

	for _, test := range testCases {

		t.Run(test.description, func(t *testing.T) {
			callWebsocket(t, test)
		})
	}
}

func callWebsocket(t *testing.T, test *testCase) {
	if test.skip {
		t.Skip("Websocket endpoint not yet implemented")
	}
	url := "ws://localhost:8546/" // todo don't hard code this
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer ws.Close()

	done := make(chan struct{})

	vals := make(chan []byte)
	go wsListener(t, ws, vals, done, len(test.expected.([]interface{})))

	err = ws.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "`+test.method+`",
    "params": [`+test.params+`],
    "id": 1
}`))
	require.NoError(t, err)
	resCount := 0
	for {
		select {
		case v := <-vals:
			resCount++
			switch exp := test.expected.([]interface{})[resCount-1].(type) {
			case int:
				// check for result subscription number
				resNum := 0
				err = rpc.Decode(v, &resNum)
				require.NoError(t, err)

			case map[string]interface{}:
				// check result map response
				resMap := make(map[string]interface{})
				err = rpc.Decode(v, &resMap)
				require.NoError(t, err)

				// check values in map are expected type
				for eKey, eVal := range exp {
					rVal := resMap[eKey]
					require.NotNil(t, rVal)
					require.IsType(t, eVal, rVal)
					switch evt := eVal.(type) {
					case map[string]interface{}:
						checkMap(t, evt, rVal.(map[string]interface{}))
					}
				}
			}

		case <-done:
			return
		}
	}
}

func wsListener(t *testing.T, ws *websocket.Conn, val chan []byte, done chan struct{}, msgCount int) {
	defer close(done)
	count := 0
	for {
		_, message, err := ws.ReadMessage()
		require.NoError(t, err)

		count++
		log.Printf("recv: %v: %s\n", count, message)

		val <- message
		if count == msgCount {
			err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			require.NoError(t, err)
			return
		}
	}
}

func checkMap(t *testing.T, expMap map[string]interface{}, ckMap map[string]interface{}) {
	for eKey, eVal := range expMap {
		cVal := ckMap[eKey]

		require.NotNil(t, cVal)
		require.IsType(t, eVal, cVal)
		switch evt := eVal.(type) {
		case map[string]interface{}:
			checkMap(t, evt, cVal.(map[string]interface{}))
		}
	}

}
