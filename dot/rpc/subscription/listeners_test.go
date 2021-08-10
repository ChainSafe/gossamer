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

package subscription

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockWSConnAPI struct {
	lastMessage BaseResponseJSON
}

func (m *MockWSConnAPI) safeSend(msg interface{}) {
	m.lastMessage = msg.(BaseResponseJSON)
}

func TestStorageObserver_Update(t *testing.T) {
	mockConnection := &MockWSConnAPI{}
	storageObserver := StorageObserver{
		id:     0,
		wsconn: mockConnection,
	}

	data := []state.KeyValue{{
		Key:   []byte("key"),
		Value: []byte("value"),
	}}
	change := &state.SubscriptionResult{
		Hash:    common.Hash{},
		Changes: data,
	}

	expected := ChangeResult{
		Block:   change.Hash.String(),
		Changes: make([]Change, len(change.Changes)),
	}
	for i, v := range change.Changes {
		expected.Changes[i] = Change{common.BytesToHex(v.Key), common.BytesToHex(v.Value)}
	}

	expectedRespones := newSubcriptionBaseResponseJSON()
	expectedRespones.Method = "state_storage"
	expectedRespones.Params.Result = expected

	storageObserver.Update(change)
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, expectedRespones, mockConnection.lastMessage)
}

func TestBlockListener_Listen(t *testing.T) {
	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	BlockAPI := new(mocks.BlockAPI)
	BlockAPI.On("UnregisterImportedChannel", mock.AnythingOfType("uint8"))

	wsconn.BlockAPI = BlockAPI

	notifyChan := make(chan *types.Block)
	bl := BlockListener{
		Channel:       notifyChan,
		wsconn:        wsconn,
		cancel:        make(chan struct{}),
		done:          make(chan struct{}),
		cancelTimeout: time.Second * 5,
	}

	block := types.NewEmptyBlock()
	block.Header.Number = big.NewInt(1)

	go bl.Listen()
	defer func() {
		require.NoError(t, bl.Stop())
		time.Sleep(time.Millisecond * 10)
		BlockAPI.AssertCalled(t, "UnregisterImportedChannel", mock.AnythingOfType("uint8"))
	}()

	notifyChan <- block
	time.Sleep(time.Second * 2)

	_, msg, err := ws.ReadMessage()
	require.NoError(t, err)

	head, err := modules.HeaderToJSON(*block.Header)
	require.NoError(t, err)

	expectedResposnse := newSubcriptionBaseResponseJSON()
	expectedResposnse.Method = chainNewHeadMethod
	expectedResposnse.Params.Result = head

	expectedResponseBytes, err := json.Marshal(expectedResposnse)
	require.NoError(t, err)

	require.Equal(t, string(expectedResponseBytes)+"\n", string(msg))
}

func TestBlockFinalizedListener_Listen(t *testing.T) {
	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	BlockAPI := new(mocks.BlockAPI)
	BlockAPI.On("UnregisterFinalisedChannel", mock.AnythingOfType("uint8"))

	wsconn.BlockAPI = BlockAPI

	notifyChan := make(chan *types.FinalisationInfo)
	bfl := BlockFinalizedListener{
		channel:       notifyChan,
		wsconn:        wsconn,
		cancel:        make(chan struct{}),
		done:          make(chan struct{}),
		cancelTimeout: time.Second * 5,
	}

	header := types.NewEmptyHeader()

	bfl.Listen()
	defer func() {
		require.NoError(t, bfl.Stop())
		time.Sleep(time.Millisecond * 10)
		BlockAPI.AssertCalled(t, "UnregisterFinalisedChannel", mock.AnythingOfType("uint8"))
	}()

	notifyChan <- &types.FinalisationInfo{
		Header: header,
	}
	time.Sleep(time.Second * 2)

	_, msg, err := ws.ReadMessage()
	require.NoError(t, err)

	head, err := modules.HeaderToJSON(*header)
	if err != nil {
		logger.Error("failed to convert header to JSON", "error", err)
	}
	expectedResponse := newSubcriptionBaseResponseJSON()
	expectedResponse.Method = chainFinalizedHeadMethod
	expectedResponse.Params.Result = head

	expectedResponseBytes, err := json.Marshal(expectedResponse)
	require.NoError(t, err)

	require.Equal(t, string(expectedResponseBytes)+"\n", string(msg))
}
func TestExtrinsicSubmitListener_Listen(t *testing.T) {
	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	notifyImportedChan := make(chan *types.Block, 100)
	notifyFinalizedChan := make(chan *types.FinalisationInfo, 100)

	BlockAPI := new(mocks.BlockAPI)
	BlockAPI.On("UnregisterImportedChannel", mock.AnythingOfType("uint8"))
	BlockAPI.On("UnregisterFinalisedChannel", mock.AnythingOfType("uint8"))

	wsconn.BlockAPI = BlockAPI

	esl := ExtrinsicSubmitListener{
		importedChan:  notifyImportedChan,
		finalisedChan: notifyFinalizedChan,
		wsconn:        wsconn,
		extrinsic:     types.Extrinsic{1, 2, 3},
		cancel:        make(chan struct{}),
		done:          make(chan struct{}),
		cancelTimeout: time.Second * 5,
	}
	header := types.NewEmptyHeader()
	exts := []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	body, err := types.NewBodyFromExtrinsics(exts)
	require.NoError(t, err)

	block := &types.Block{
		Header: header,
		Body:   body,
	}

	esl.Listen()
	defer func() {
		require.NoError(t, esl.Stop())
		time.Sleep(time.Millisecond * 10)

		BlockAPI.AssertCalled(t, "UnregisterImportedChannel", mock.AnythingOfType("uint8"))
		BlockAPI.AssertCalled(t, "UnregisterFinalisedChannel", mock.AnythingOfType("uint8"))
	}()

	notifyImportedChan <- block
	time.Sleep(time.Second * 2)

	_, msg, err := ws.ReadMessage()
	require.NoError(t, err)
	resImported := map[string]interface{}{"inBlock": block.Header.Hash().String()}
	expectedImportedBytes, err := json.Marshal(newSubscriptionResponse(authorExtrinsicUpdatesMethod, esl.subID, resImported))
	require.NoError(t, err)
	require.Equal(t, string(expectedImportedBytes)+"\n", string(msg))

	notifyFinalizedChan <- &types.FinalisationInfo{
		Header: header,
	}
	time.Sleep(time.Second * 2)

	_, msg, err = ws.ReadMessage()
	require.NoError(t, err)
	resFinalised := map[string]interface{}{"finalised": block.Header.Hash().String()}
	expectedFinalizedBytes, err := json.Marshal(newSubscriptionResponse(authorExtrinsicUpdatesMethod, esl.subID, resFinalised))
	require.NoError(t, err)
	require.Equal(t, string(expectedFinalizedBytes)+"\n", string(msg))
}

func TestGrandpaJustification_Listen(t *testing.T) {
	t.Run("When justification doesnt returns error", func(t *testing.T) {
		wsconn, ws, cancel := setupWSConn(t)
		defer cancel()

		mockedJust := grandpa.Justification{
			Round: 1,
			Commit: &grandpa.Commit{
				Hash:       common.Hash{},
				Number:     1,
				Precommits: nil,
			},
		}

		mockedJustBytes, err := mockedJust.Encode()
		require.NoError(t, err)

		blockStateMock := new(mocks.BlockAPI)
		blockStateMock.On("GetJustification", mock.AnythingOfType("common.Hash")).Return(mockedJustBytes, nil)
		blockStateMock.On("UnregisterFinalisedChannel", mock.AnythingOfType("uint8"))
		wsconn.BlockAPI = blockStateMock

		finchannel := make(chan *types.FinalisationInfo)
		sub := GrandpaJustificationListener{
			subID:         10,
			wsconn:        wsconn,
			cancel:        make(chan struct{}, 1),
			done:          make(chan struct{}, 1),
			finalisedCh:   finchannel,
			cancelTimeout: time.Second * 5,
		}

		sub.Listen()
		finchannel <- &types.FinalisationInfo{
			Header: types.NewEmptyHeader(),
		}

		time.Sleep(time.Second * 3)

		_, msg, err := ws.ReadMessage()
		require.NoError(t, err)

		expected := `{"jsonrpc":"2.0","method":"grandpa_justifications","params":{"result":"%s","subscription":10}}` + "\n"
		expected = fmt.Sprintf(expected, common.BytesToHex(mockedJustBytes))

		require.Equal(t, string(msg), expected)
		require.NoError(t, sub.Stop())
		wsconn.Wsconn.Close()
	})
}

func setupWSConn(t *testing.T) (*WSConn, *websocket.Conn, func()) {
	t.Helper()

	wskt := new(WSConn)
	var up = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			log.Print("error while setup handler:", err)
			return
		}

		wskt.Wsconn = c
	}

	server := httptest.NewServer(http.HandlerFunc(h))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, r, err := websocket.DefaultDialer.Dial(wsURL, nil)
	defer r.Body.Close()

	require.NoError(t, err)

	cancel := func() {
		server.Close()
		ws.Close()
		wskt.Wsconn.Close()
	}

	return wskt, ws, cancel
}
