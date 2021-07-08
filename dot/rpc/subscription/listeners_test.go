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
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
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
	notifyChan := make(chan *types.Block)
	mockConnection := &MockWSConnAPI{}
	bl := BlockListener{
		Channel: notifyChan,
		wsconn:  mockConnection,
	}

	block := types.NewEmptyBlock()
	block.Header.Number = big.NewInt(1)

	head, err := modules.HeaderToJSON(*block.Header)
	require.NoError(t, err)

	expectedResposnse := newSubcriptionBaseResponseJSON()
	expectedResposnse.Method = "chain_newHead"
	expectedResposnse.Params.Result = head

	go bl.Listen()

	notifyChan <- block
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, expectedResposnse, mockConnection.lastMessage)
}

func TestBlockFinalizedListener_Listen(t *testing.T) {
	notifyChan := make(chan *types.FinalisationInfo)
	mockConnection := &MockWSConnAPI{}
	bfl := BlockFinalizedListener{
		channel: notifyChan,
		wsconn:  mockConnection,
	}

	header := types.NewEmptyHeader()
	head, err := modules.HeaderToJSON(*header)
	if err != nil {
		logger.Error("failed to convert header to JSON", "error", err)
	}
	expectedResponse := newSubcriptionBaseResponseJSON()
	expectedResponse.Method = "chain_finalizedHead"
	expectedResponse.Params.Result = head

	go bfl.Listen()

	notifyChan <- &types.FinalisationInfo{
		Header: header,
	}
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, expectedResponse, mockConnection.lastMessage)
}

func TestExtrinsicSubmitListener_Listen(t *testing.T) {
	notifyImportedChan := make(chan *types.Block, 100)
	notifyFinalizedChan := make(chan *types.FinalisationInfo, 100)

	mockConnection := &MockWSConnAPI{}
	esl := ExtrinsicSubmitListener{
		importedChan:  notifyImportedChan,
		finalisedChan: notifyFinalizedChan,
		wsconn:        mockConnection,
		extrinsic:     types.Extrinsic{1, 2, 3},
	}
	header := types.NewEmptyHeader()
	exts := []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	body, err := types.NewBodyFromExtrinsics(exts)
	require.NoError(t, err)

	block := &types.Block{
		Header: header,
		Body:   body,
	}

	resImported := map[string]interface{}{"inBlock": block.Header.Hash().String()}
	expectedImportedRespones := newSubscriptionResponse(AuthorExtrinsicUpdates, esl.subID, resImported)

	go esl.Listen()

	notifyImportedChan <- block
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, expectedImportedRespones, mockConnection.lastMessage)

	notifyFinalizedChan <- &types.FinalisationInfo{
		Header: header,
	}
	time.Sleep(time.Millisecond * 10)
	resFinalised := map[string]interface{}{"finalised": block.Header.Hash().String()}
	expectedFinalizedRespones := newSubscriptionResponse(AuthorExtrinsicUpdates, esl.subID, resFinalised)
	require.Equal(t, expectedFinalizedRespones, mockConnection.lastMessage)
}
