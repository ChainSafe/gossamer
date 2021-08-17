// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package network

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDecodeTransactionHandshake(t *testing.T) {
	testHandshake := &transactionHandshake{}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)

	msg, err := decodeTransactionHandshake(enc)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)
}

func TestDecodeTransactionMessage(t *testing.T) {
	testTxMsg := &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}

	enc, err := testTxMsg.Encode()
	require.NoError(t, err)

	msg, err := decodeTransactionMessage(enc)
	require.NoError(t, err)
	require.Equal(t, testTxMsg, msg)
}

func TestHandleTransactionMessage(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	mockhandler := &MockTransactionHandler{}
	mockhandler.On("HandleTransactionMessage", mock.AnythingOfType("*network.TransactionMessage")).Return(true, nil)
	mockhandler.On("TransactionsCount").Return(0)

	config := &Config{
		BasePath:           basePath,
		Port:               7001,
		NoBootstrap:        true,
		NoMDNS:             true,
		TransactionHandler: mockhandler,
	}

	s := createTestService(t, config)

	msg := &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}

	s.handleTransactionMessage(peer.ID(""), msg)
	mockhandler.AssertCalled(t, "HandleTransactionMessage", msg)
}

//PrintMemUsage outputs the current, total and OS memory being used. As well as the number
//of garage collection cycles completed.
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func TestDecodeTransactionMessageEncode(t *testing.T) {
	testTxMsg := TransactionMessage{
		Extrinsics: []types.Extrinsic{}, // Store large data
	}
	for i := 0; i <= 1000; i++ {
		token := make([]byte, 100000)
		rand.Read(token)
		testTxMsg.Extrinsics = append(testTxMsg.Extrinsics, token)
	}
	enc, err := testTxMsg.EncodeOld()
	require.NoError(t, err)
	PrintMemUsage()

	_ = enc
	// Force GC to clear up, should see a memory drop
	runtime.GC()

	encNew, err := testTxMsg.Encode()
	require.NoError(t, err)
	PrintMemUsage()
	_ = encNew
}

func TestDecodeTransactionMessageEncodeRandom(t *testing.T) {
	testTxMsg := TransactionMessage{
		Extrinsics: []types.Extrinsic{}, // Store large data
	}
	for i := 0; i < 5; i++ {
		token := make([]byte, 10)
		//rand.Read(token)
		testTxMsg.Extrinsics = append(testTxMsg.Extrinsics, token)
	}
	enc, err := testTxMsg.Encode()
	require.NoError(t, err)

	encScale, err := scale.Marshal(testTxMsg.Extrinsics)
	require.NoError(t, err)

	require.Equal(t, enc, encScale)
	fmt.Println(common.BytesToHex(encScale))
}
