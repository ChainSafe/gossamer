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

package babe

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
)

func newTestVerificationManager(t *testing.T) *VerificationManager {
	dbSrv := state.NewService("")
	dbSrv.UseMemDB()

	genesisData := new(genesis.Data)

	err := dbSrv.Initialize(genesisData, genesisHeader, trie.NewEmptyTrie())
	if err != nil {
		t.Fatal(err)
	}

	err = dbSrv.Start()
	if err != nil {
		t.Fatal(err)
	}

	vm, err := NewVerificationManager(dbSrv.Block, 0, &NextEpochDescriptor{})
	if err != nil {
		t.Fatal(err)
	}

	return vm
}

// // test getBlockEpoch
// func TestGetBlockEpoch(t *testing.T) {
// 	vm := newTestVerificationManager(t)

// 	blockHash := s.blockState.BestBlockHash()

// 	epoch, err := s.getBlockEpoch(blockHash)
// 	require.Nil(t, err)

// 	require.Equal(t, s.currentEpoch(), epoch)
// }

// // test isBlockFromEpoch
// func TestIsBlockFromEpoch(t *testing.T) {
// 	s := newTestSyncer(t, nil)
// 	addTestBlocksToState(t, 1, s.blockState)

// 	s.verificationManager.SetCurrentEpoch(2)

// 	blockHash := s.blockState.BestBlockHash()

// 	currentEpoch, err := s.blockFromCurrentEpoch(blockHash)
// 	require.Nil(t, err)

// 	require.Equal(t, true, currentEpoch)
// }

func TestCheckForConsensusDigest(t *testing.T) {

}

func TestVerifySlotWinner(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &SessionConfig{
		Keypair: kp,
	}

	babesession := createTestSession(t, cfg)
	err = babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	// create proof that we can authorize this block
	babesession.epochThreshold = big.NewInt(0)
	babesession.authorityIndex = 0
	var slotNumber uint64 = 1

	addAuthorshipProof(t, babesession, slotNumber)

	slot := Slot{
		start:    uint64(time.Now().Unix()),
		duration: uint64(10000000),
		number:   slotNumber,
	}

	// create babe header
	babeHeader, err := babesession.buildBlockBabeHeader(slot)
	if err != nil {
		t.Fatal(err)
	}

	authorityData := make([]*AuthorityData, 1)
	authorityData[0] = &AuthorityData{
		ID: kp.Public().(*sr25519.PublicKey),
	}

	verifier, err := NewVerifier(babesession.blockState, &NextEpochDescriptor{
		Authorities: babesession.authorityData,
		Randomness:  [32]byte{babesession.config.Randomness},
	})

	if err != nil {
		t.Fatal(err)
	}

	ok, err := verifier.verifySlotWinner(slot.number, babeHeader)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("did not verify slot winner")
	}
}

func TestVerifyAuthorshipRight(t *testing.T) {
	babesession := createTestSession(t, nil)
	err := babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	// see https://github.com/noot/substrate/blob/add-blob/core/test-runtime/src/system.rs#L468
	txb := []byte{3, 16, 110, 111, 111, 116, 1, 64, 103, 111, 115, 115, 97, 109, 101, 114, 95, 105, 115, 95, 99, 111, 111, 108}

	block, _ := createTestBlock(t, babesession, [][]byte{txb})

	verifier, err := NewVerifier(babesession.blockState, &NextEpochDescriptor{
		Authorities: babesession.authorityData,
		Randomness:  [32]byte{babesession.config.Randomness},
	})
	if err != nil {
		t.Fatal(err)
	}

	ok, err := verifier.verifyAuthorshipRight(block.Header)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("did not verify authorship right")
	}
}

func TestVerifyAuthorshipRight_Equivocation(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &SessionConfig{
		Keypair: kp,
	}

	babesession := createTestSession(t, cfg)
	err = babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	babesession.authorityData = make([]*AuthorityData, 1)
	babesession.authorityData[0] = &AuthorityData{
		ID: kp.Public().(*sr25519.PublicKey),
	}

	// create and add first block
	block, _ := createTestBlock(t, babesession, [][]byte{})
	block.Header.Hash()

	err = babesession.blockState.AddBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	verifier, err := NewVerifier(babesession.blockState, &NextEpochDescriptor{
		Authorities: babesession.authorityData,
		Randomness:  [32]byte{babesession.config.Randomness},
	})
	if err != nil {
		t.Fatal(err)
	}

	ok, err := verifier.verifyAuthorshipRight(block.Header)
	require.NoError(t, err)
	require.True(t, ok)

	// create new block
	// see https://github.com/noot/substrate/blob/add-blob/core/test-runtime/src/system.rs#L468
	txb := []byte{3, 16, 110, 111, 111, 116, 1, 64, 103, 111, 115, 115, 97, 109, 101, 114, 95, 105, 115, 95, 99, 111, 111, 108}

	block2, _ := createTestBlock(t, babesession, [][]byte{txb})
	block2.Header.Hash()

	t.Log(block2.Header)

	err = babesession.blockState.AddBlock(block2)
	if err != nil {
		t.Fatal(err)
	}

	ok, err = verifier.verifyAuthorshipRight(block2.Header)
	require.NotNil(t, err)
	require.False(t, ok)
	require.Equal(t, ErrProducerEquivocated, err)
}
