// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
//go:build integration

package subscription

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWSConn_HandleConnParallel(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		badRequest             bool
		msg                    []byte
		subscriptions          int
		initErr                error
		readErrMsg, initErrMsg string
		storageAPIset          bool
		reqID                  float64
		params                 interface{}
	}{
		"test_StorageAPI_not_set": {
			badRequest: true,
			msg: []byte(`{"jsonrpc":"2.0",` +
				`"error":{"code":null,"message":"error StorageAPI not set"},` +
				`"id":1}` + "\n"),
			subscriptions: 0,
			initErr:       errStorageNotSet,
			initErrMsg:    "error StorageAPI not set",
			storageAPIset: false,
			reqID:         1,
			params:        nil,
		},
		"req_1_unexpected_type_nil": {
			badRequest:    true,
			msg:           nil,
			subscriptions: 0,
			initErr:       errUnexpectedType,
			initErrMsg:    "unexpected type: <nil>, expected type []interface{}",
			storageAPIset: true,
			reqID:         1,
			params:        nil,
		},
		"req_2": {
			badRequest:    false,
			msg:           []byte(`{"jsonrpc":"2.0","result":1,"id":2}` + "\n"),
			subscriptions: 1,
			initErr:       nil,
			initErrMsg:    "",
			storageAPIset: true,
			reqID:         2,
			params:        []interface{}{},
		},
		"req_3": {
			badRequest:    false,
			msg:           []byte(`{"jsonrpc":"2.0","result":1,"id":3}` + "\n"),
			subscriptions: 1,
			initErr:       nil,
			initErrMsg:    "",
			storageAPIset: true,
			reqID:         3,
			params:        []interface{}{"0x26aa"},
		},
		"req_1_unexpected_type_[]int": {
			badRequest:    true,
			msg:           nil,
			subscriptions: 0,
			initErr:       errUnexpectedType,
			initErrMsg:    "unexpected type: []int, expected type string, []string, []interface{}",
			storageAPIset: true,
			reqID:         1,
			params:        []interface{}{[]int{123}},
		},
		"req_1_unexpected_type_int": {
			badRequest:    true,
			msg:           nil,
			subscriptions: 0,
			initErr:       errUnexpectedType,
			initErrMsg:    "unexpected type: int, expected type string, []string, []interface{}",
			storageAPIset: true,
			reqID:         1,
			params:        []interface{}{123},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			wsconn, c, cancel := setupWSConn(t)
			defer cancel()

			wsconn.Subscriptions = make(map[uint32]Listener)

			if testCase.storageAPIset {
				wsconn.StorageAPI = modules.NewMockAnyStorageAPI(ctrl)
			}

			go wsconn.HandleConn()
			time.Sleep(time.Second * 2)

			res, initErr := wsconn.initStorageChangeListener(testCase.reqID, testCase.params)

			if testCase.badRequest {
				require.Nil(t, res)
			} else {
				require.NotNil(t, res)
			}
			require.Equal(t, len(wsconn.Subscriptions), testCase.subscriptions)
			require.ErrorIs(t, initErr, testCase.initErr)
			if testCase.initErr != nil {
				require.EqualError(t, initErr, testCase.initErrMsg)
			}

			if testCase.msg != nil {
				_, msg, readErr := c.ReadMessage()
				require.NoError(t, readErr)
				require.Equal(t, testCase.msg, msg)
			}

		})
	}
}

func TestWSConn_HandleConnSubscriptionsIncrement(t *testing.T) {
	t.Parallel()
	type Case struct {
		reqID         float64
		params        []interface{}
		response      []byte
		subscriptions int
	}
	testCases := map[string]struct {
		requests []*Case
	}{
		"test_case_with_1_request": {
			requests: []*Case{
				{
					reqID:         1,
					params:        []interface{}{"0x26aa"},
					subscriptions: 1,
					response:      []byte(`{"jsonrpc":"2.0","result":1,"id":1}` + "\n"),
				},
			},
		},
		"test_case_with_2_requests": {
			requests: []*Case{
				{
					reqID:         1,
					params:        []interface{}{"0x26aa"},
					subscriptions: 1,
					response:      []byte(`{"jsonrpc":"2.0","result":1,"id":1}` + "\n"),
				},
				{
					reqID:         2,
					params:        []interface{}{"0x26ab"},
					subscriptions: 2,
					response:      []byte(`{"jsonrpc":"2.0","result":2,"id":2}` + "\n"),
				},
			},
		},
		"test_case_with_4_requests_1_bad": {
			requests: []*Case{
				{
					reqID:         1,
					params:        []interface{}{"0x26aa"},
					subscriptions: 1,
					response:      []byte(`{"jsonrpc":"2.0","result":1,"id":1}` + "\n"),
				},
				{
					reqID:         2,
					params:        []interface{}{"0x26ab"},
					subscriptions: 2,
					response:      []byte(`{"jsonrpc":"2.0","result":2,"id":2}` + "\n"),
				},
				{
					reqID:         3,
					params:        []interface{}{[]int{123}},
					subscriptions: 2,
					response:      nil,
				},
				{
					reqID:         4,
					params:        []interface{}{"0x26aa"},
					subscriptions: 3,
					response:      []byte(`{"jsonrpc":"2.0","result":3,"id":4}` + "\n"),
				},
			},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			wsconn, c, cancel := setupWSConn(t)
			wsconn.Subscriptions = make(map[uint32]Listener)
			wsconn.StorageAPI = modules.NewMockAnyStorageAPI(ctrl)
			defer cancel()

			go wsconn.HandleConn()
			time.Sleep(time.Second * 2)

			for _, v := range testCase.requests {
				res, err := wsconn.initStorageChangeListener(v.reqID, v.params)
				// bad request
				if v.response == nil {
					require.Nil(t, res)
					require.Error(t, err)
				} else {
					require.NotNil(t, res)
					require.NoError(t, err)
				}
				require.Len(t, wsconn.Subscriptions, v.subscriptions)
				_, msg, err := c.ReadMessage()
				require.NoError(t, err)
				require.Equal(t, v.response, msg)
			}
		})
	}
}

func TestWSCon_WriteMessage(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		request []byte
		respErr error
		resp    []byte
	}{
		"state_subscribeStorage_request": {
			request: []byte(`{
    		"jsonrpc": "2.0",
			"method": "state_subscribeStorage",
    		"params": ["` +
				`0x26aa394eea5630e07c48ae0c9558c` +
				`ef7b99d880ec681799c0cf30e888637` +
				`1da9de1e86a9a8c739864cf3cc5ec2b` +
				`ea59fd43593c715fdd31c61141abd04` +
				`a99fd6822c8558854ccde39a5684e7a` +
				`56da27d"],
			"id": 7}`),
			respErr: nil,
			resp:    []byte(`{"jsonrpc":"2.0","result":1,"id":7}` + "\n"),
		},
		"invalid_state_unsubscribeStorage_request_wrong_params": {
			request: []byte(`{
    					"jsonrpc": "2.0",
						"method": "state_unsubscribeStorage",
						"params": "foo",
						"id": 7}`),
			respErr: nil,
			resp:    []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":7}` + "\n"),
		},
		"invalid_state_unsubscribeStorage_request_empty_params_array": {
			request: []byte(`{
    					"jsonrpc": "2.0",
						"method": "state_unsubscribeStorage",
						"params": "[]",
						"id": 7}`),
			respErr: nil,
			resp:    []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":7}` + "\n"),
		},
		"state_unsubscribeStorage_request_#1_string_param": {
			request: []byte(`{
    					"jsonrpc": "2.0",
						"method": "state_unsubscribeStorage",
						"params": ["6"],
						"id": 7}`),
			respErr: nil,
			resp:    []byte(`{"jsonrpc":"2.0","result":false,"id":7}` + "\n"),
		},
		"state_unsubscribeStorage_request_#2_string_param_": {
			request: []byte(`{
    					"jsonrpc": "2.0",
						"method": "state_unsubscribeStorage",
						"params": ["4"],
						"id": 7}`),
			respErr: nil,
			resp:    []byte(`{"jsonrpc":"2.0","result":true,"id":7}` + "\n"),
		},
		"state_unsubscribeStorage_request_#3_int_param": {
			request: []byte(`{
    					"jsonrpc": "2.0",
						"method": "state_unsubscribeStorage",
						"params": [6],
						"id": 7}`),
			respErr: nil,
			resp:    []byte(`{"jsonrpc":"2.0","result":false,"id":7}` + "\n"),
		},
		"state_unsubscribeStorage_request_#4_int_param": {
			request: []byte(`{
    					"jsonrpc": "2.0",
						"method": "state_unsubscribeStorage",
						"params": [4],
						"id": 7}`),
			respErr: nil,
			resp:    []byte(`{"jsonrpc":"2.0","result":true,"id":7}` + "\n"),
		},
		"chain_subscribeNewHeads_request": {
			request: []byte(`{
							"jsonrpc": "2.0",
							"method": "chain_subscribeNewHeads",
							"params": [],
							"id": 8
						}`),
			respErr: nil,
			resp:    []byte(`{"jsonrpc":"2.0","result":6,"id":8}` + "\n"),
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			wsconn, c, cancel := setupWSConn(t)
			defer cancel()

			wsconn.Subscriptions = make(map[uint32]Listener)
			wsconn.StorageAPI = modules.NewMockAnyStorageAPI(ctrl)

			go wsconn.HandleConn()
			time.Sleep(time.Second * 2)

			err := c.WriteMessage(websocket.TextMessage, testCase.request)
			require.NoError(t, err)

			_, msg, err := c.ReadMessage()
			require.NoError(t, err)
			require.Equal(t, testCase.resp, msg)
		})
	}

}

func TestWSConn_InitBlockListener(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		setBlocAPI bool
		respErr    error
		resp       []byte
	}{
		"blockAPI_not_set": {
			setBlocAPI: false,
			resp:       []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error BlockAPI not set"},"id":1}` + "\n"),
			respErr:    errors.New("error BlockAPI not set"),
		},
		"blockAPI_set": {
			setBlocAPI: true,
			resp:       []byte(`{"jsonrpc":"2.0","result":1,"id":1}` + "\n"),
			respErr:    nil,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			wsconn, c, cancel := setupWSConn(t)
			defer cancel()

			wsconn.Subscriptions = make(map[uint32]Listener)
			wsconn.StorageAPI = modules.NewMockAnyStorageAPI(ctrl)

			go wsconn.HandleConn()
			time.Sleep(time.Second * 2)

			if testCase.setBlocAPI {
				wsconn.BlockAPI = modules.NewMockAnyBlockAPI(ctrl)
			}

			res, err := wsconn.initBlockListener(1, nil)

			if testCase.respErr != nil {
				require.EqualError(t, err, testCase.respErr.Error())
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
			}

			_, msg, err := c.ReadMessage()
			require.NoError(t, err)
			require.Equal(t, testCase.resp, msg)

		})
	}
}

func TestWSConn_InitBlockFinalizedListener(t *testing.T) {
	ctrl := gomock.NewController(t)

	wsconn, c, cancel := setupWSConn(t)
	wsconn.Subscriptions = make(map[uint32]Listener)
	defer cancel()

	go wsconn.HandleConn()
	time.Sleep(time.Second * 2)

	wsconn.StorageAPI = modules.NewMockAnyStorageAPI(ctrl)

	wsconn.BlockAPI = nil

	res, err := wsconn.initBlockFinalizedListener(1, nil)
	require.EqualError(t, err, "error BlockAPI not set")
	require.Nil(t, res)
	_, msg, err := c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error BlockAPI not set"},"id":1}`+"\n"), msg)

	wsconn.BlockAPI = modules.NewMockAnyBlockAPI(ctrl)

	res, err = wsconn.initBlockFinalizedListener(1, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, wsconn.Subscriptions, 1)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":1,"id":1}`+"\n"), msg)
}

func TestWSConn_InitExtrinsicWatchTest(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		setBlocAPI bool
		initErr    error
		reqID      float64
		param      []interface{}
		msg        []byte
	}{
		"non_hex_params": {
			setBlocAPI: false,
			initErr:    errors.New("could not byteify non 0x prefixed string: NotHex"),
			reqID:      0,
			param:      []interface{}{"NotHex"},
		},
		"block_API_not_set": {
			setBlocAPI: false,
			initErr:    errors.New("could not byteify non 0x prefixed string: NotHex"),
			reqID:      0,
			param:      []interface{}{"NotHex"},
		},
		"block_API_set": {
			setBlocAPI: true,
			initErr:    nil,
			reqID:      0,
			param:      []interface{}{"0x26aa"},
			msg:        []byte(`{"jsonrpc":"2.0","result":1,"id":0}` + "\n"),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			wsconn, c, cancel := setupWSConn(t)
			wsconn.Subscriptions = make(map[uint32]Listener)
			defer cancel()

			go wsconn.HandleConn()
			time.Sleep(time.Second * 2)

			wsconn.StorageAPI = modules.NewMockAnyStorageAPI(ctrl)
			wsconn.CoreAPI = modules.NewMockAnyAPI(ctrl)
			if testCase.setBlocAPI {
				wsconn.BlockAPI = modules.NewMockAnyBlockAPI(ctrl)
				transactionStateAPI := NewMockTransactionStateAPI(ctrl)
				transactionStateAPI.EXPECT().GetStatusNotifierChannel(gomock.Any()).Return(make(chan transaction.Status)).Times(1)
				wsconn.TxStateAPI = transactionStateAPI
			}

			listner, err := wsconn.initExtrinsicWatch(testCase.reqID, testCase.param)
			if testCase.setBlocAPI {
				require.NotNil(t, listner)
				require.NoError(t, err)
				_, msg, err := c.ReadMessage()
				require.NoError(t, err)
				require.Equal(t, testCase.msg, msg)
			} else {
				require.Nil(t, listner)
				require.EqualError(t, err, testCase.initErr.Error())
			}
		})
	}

}
func TestWSConn_InitExtrinsicWatch(t *testing.T) {
	ctrl := gomock.NewController(t)

	wsconn, c, cancel := setupWSConn(t)
	wsconn.Subscriptions = make(map[uint32]Listener)
	defer cancel()

	go wsconn.HandleConn()
	time.Sleep(time.Second * 2)

	wsconn.StorageAPI = modules.NewMockAnyStorageAPI(ctrl)
	wsconn.BlockAPI = modules.NewMockAnyBlockAPI(ctrl)
	transactionStateAPI := NewMockTransactionStateAPI(ctrl)
	transactionStateAPI.EXPECT().GetStatusNotifierChannel(gomock.Any()).Return(make(chan transaction.Status)).Times(1)
	wsconn.TxStateAPI = transactionStateAPI

	// test initExtrinsicWatch with invalid transaction
	invalidTransaction := runtime.NewInvalidTransaction()
	err := invalidTransaction.Set(runtime.Future{})
	require.NoError(t, err)
	coreAPI := mocks.NewMockCoreAPI(ctrl)
	wsconn.CoreAPI = coreAPI
	coreAPI.EXPECT().HandleSubmittedExtrinsic(gomock.Any()).Return(invalidTransaction)
	listner, err := wsconn.initExtrinsicWatch(0, []interface{}{"0xa9018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d019e91c8d44bf01ffe36d54f9e43dade2b2fc653270a0e002daed1581435c2e1755bc4349f1434876089d99c9dac4d4128e511c2a3e0788a2a74dd686519cb7c83000000000104ab"}) //nolint:lll
	require.Error(t, err)
	require.Nil(t, listner)

	_, msg, err := c.ReadMessage()

	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","method":"author_extrinsicUpdate",`+
		`"params":{"result":"invalid","subscription":1}}`+"\n", string(msg))

	mockedJust := grandpa.Justification{
		Round: 1,
		Commit: grandpa.Commit{
			Number:     1,
			Precommits: nil,
		},
	}

	mockedJustBytes, err := scale.Marshal(mockedJust)
	require.NoError(t, err)

	wsconn.CoreAPI = modules.NewMockAnyAPI(ctrl)
	BlockAPI := mocks.NewMockBlockAPI(ctrl)

	fCh := make(chan *types.FinalisationInfo, 5)
	BlockAPI.EXPECT().GetFinalisedNotifierChannel().Return(fCh)

	BlockAPI.EXPECT().GetJustification(gomock.Any()).Return(mockedJustBytes, nil)
	BlockAPI.EXPECT().FreeFinalisedNotifierChannel(gomock.Any())

	wsconn.BlockAPI = BlockAPI
	listener, err := wsconn.initGrandpaJustificationListener(0, nil)
	require.NoError(t, err)
	require.NotNil(t, listener)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","result":2,"id":0}`+"\n", string(msg))

	listener.Listen()
	header := &types.Header{
		Number: 1,
	}

	fCh <- &types.FinalisationInfo{
		Header: *header,
	}

	time.Sleep(time.Second * 2)

	expected := `{"jsonrpc":"2.0","method":"grandpa_justifications","params":{"result":"%s","subscription":2}}` + "\n"
	expected = fmt.Sprintf(expected, common.BytesToHex(mockedJustBytes))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(expected), msg)

	err = listener.Stop()
	require.NoError(t, err)
}

func TestSubscribeAllHeads(t *testing.T) {
	ctrl := gomock.NewController(t)

	wsconn, c, cancel := setupWSConn(t)
	wsconn.Subscriptions = make(map[uint32]Listener)
	defer cancel()

	go wsconn.HandleConn()
	time.Sleep(time.Second * 2)

	_, err := wsconn.initAllBlocksListerner(1, nil)
	require.EqualError(t, err, "error BlockAPI not set")
	_, msg, err := c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error BlockAPI not set"},"id":1}`+"\n"), msg)

	mockBlockAPI := mocks.NewMockBlockAPI(ctrl)

	wsconn.BlockAPI = mockBlockAPI

	iCh := make(chan *types.Block)
	mockBlockAPI.EXPECT().GetImportedBlockNotifierChannel().Return(iCh)

	fCh := make(chan *types.FinalisationInfo)
	mockBlockAPI.EXPECT().GetFinalisedNotifierChannel().Return(fCh)

	l, err := wsconn.initAllBlocksListerner(1, nil)
	require.NoError(t, err)
	require.NotNil(t, l)
	require.IsType(t, &AllBlocksListener{}, l)
	require.Len(t, wsconn.Subscriptions, 1)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":1,"id":1}`+"\n"), msg)

	l.Listen()
	time.Sleep(time.Millisecond * 500)

	expected := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"chain_allHead",`+
			`"params":{"result":{"parentHash":"%s","number":"0x00",`+
			`"stateRoot":"%s","extrinsicsRoot":"%s",`+
			`"digest":{"logs":["0x064241424504ff"]}},"subscription":1}}`,
		common.Hash{},
		common.Hash{},
		common.Hash{},
	)

	digest := types.NewDigest()
	err = digest.Add(*types.NewBABEPreRuntimeDigest([]byte{0xff}))
	require.NoError(t, err)
	fCh <- &types.FinalisationInfo{
		Header: types.Header{
			Number: 0,
			Digest: digest,
		},
	}

	time.Sleep(time.Millisecond * 500)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, expected+"\n", string(msg))

	digest = types.NewDigest()
	err = digest.Add(*types.NewBABEPreRuntimeDigest([]byte{0xff}))
	require.NoError(t, err)

	iCh <- &types.Block{
		Header: types.Header{
			Number: 0,
			Digest: digest,
		},
	}
	time.Sleep(time.Millisecond * 500)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(expected+"\n"), msg)

	mockBlockAPI.EXPECT().FreeImportedBlockNotifierChannel(gomock.Any())
	mockBlockAPI.EXPECT().FreeFinalisedNotifierChannel(gomock.Any())

	require.NoError(t, l.Stop())
}

func TestWSConn_CheckWebsocketInvalidData(t *testing.T) {
	wsconn, c, cancel := setupWSConn(t)
	wsconn.Subscriptions = make(map[uint32]Listener)
	defer cancel()

	go wsconn.HandleConn()

	tests := []struct {
		sentMessage []byte
		expected    []byte
	}{
		{
			sentMessage: []byte(`{
			"jsonrpc": "2.0",
			"method": "",
			"id": 0,
			"params": []
			}`),
			expected: []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":0}` + "\n"),
		},
		{
			sentMessage: []byte(`{
			"jsonrpc": "2.0",
			"params": []
			}`),
			expected: []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":0}` + "\n"),
		},
		{
			sentMessage: []byte(`{
			"jsonrpc": "2.0",
			"id": "abcdef"
			"method": "some_method_name"
			"params": []
			}`),
			expected: []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":0}` + "\n"),
		},
	}

	for _, tt := range tests {
		c.WriteMessage(websocket.TextMessage, tt.sentMessage)

		_, msg, err := c.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, tt.expected, msg)
	}
}
