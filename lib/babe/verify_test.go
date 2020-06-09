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
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
)

var testEpoch = uint64(2)

func newTestVerificationManager(t *testing.T, withBlock bool, descriptor *NextEpochDescriptor) *VerificationManager {
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

	if descriptor == nil {
		descriptor = &NextEpochDescriptor{}
	}

	// currentEpoch = 2
	vm, err := NewVerificationManager(dbSrv.Block, testEpoch, descriptor)
	if err != nil {
		t.Fatal(err)
	}

	if withBlock {
		// preDigest with slot in epoch testEpoch = 2
		// TODO: use BABE functions to do calculate pre-digest dynamically
		preDigest, err := common.HexToBytes("0x064241424538e93dcef2efc275b72b4fa748332dc4c9f13be1125909cf90c8e9109c45da16b04bc5fdf9fe06a4f35e4ae4ed7e251ff9ee3d0d840c8237c9fb9057442dbf00f210d697a7b4959f792a81b948ff88937e30bf9709a8ab1314f71284da89a40000000000000000001100000000000000")
		require.Nil(t, err)

		nextEpochData := &NextEpochDescriptor{
			Authorities: []*types.BABEAuthorityData{},
		}

		consensusDigest := &types.ConsensusDigest{
			ConsensusEngineID: types.BabeEngineID,
			Data:              nextEpochData.Encode(),
		}

		conDigest := consensusDigest.Encode()

		header := &types.Header{
			ParentHash: genesisHeader.Hash(),
			Number:     big.NewInt(1),
			Digest:     [][]byte{preDigest, conDigest},
		}

		firstBlock := &types.Block{
			Header: header,
			Body:   &types.Body{},
		}

		err = vm.blockState.AddBlock(firstBlock)
		require.Nil(t, err)
	}

	return vm
}

// test getBlockEpoch
func TestGetBlockEpoch(t *testing.T) {
	vm := newTestVerificationManager(t, true, nil)

	header, err := vm.blockState.BestBlockHeader()
	require.Nil(t, err)

	epoch, err := vm.getBlockEpoch(header)
	require.Nil(t, err)

	require.Equal(t, vm.currentEpoch, epoch)
}

// test isBlockFromEpoch
func TestIsBlockFromEpoch(t *testing.T) {
	vm := newTestVerificationManager(t, true, nil)

	header, err := vm.blockState.BestBlockHeader()
	require.Nil(t, err)

	ok, err := vm.isBlockFromEpoch(header, testEpoch)
	require.Nil(t, err)
	require.Equal(t, true, ok)

	ok, err = vm.isBlockFromEpoch(header, 1)
	require.Nil(t, err)
	require.Equal(t, false, ok)
}

func TestCheckForConsensusDigest_NoDigest(t *testing.T) {
	header := &types.Header{
		ParentHash: genesisHeader.Hash(),
		Number:     big.NewInt(1),
	}

	_, err := checkForConsensusDigest(header)
	require.NotNil(t, err)
}

func TestCheckForConsensusDigest_NoConsensusDigest(t *testing.T) {
	vm := newTestVerificationManager(t, true, nil)

	header, err := vm.blockState.BestBlockHeader()
	require.Nil(t, err)

	header.Digest = header.Digest[:1]

	digest, err := checkForConsensusDigest(header)
	require.Nil(t, err)
	require.Nil(t, digest)
}

func TestCheckForConsensusDigest(t *testing.T) {
	vm := newTestVerificationManager(t, true, nil)

	header, err := vm.blockState.BestBlockHeader()
	require.Nil(t, err)

	digest, err := checkForConsensusDigest(header)
	require.Nil(t, err)

	nextEpochData := &NextEpochDescriptor{
		Authorities: []*types.BABEAuthorityData{},
	}

	expected := &types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              nextEpochData.Encode(),
	}

	require.Equal(t, expected, digest)
}

func TestVerificationManager_VerifyBlock(t *testing.T) {
	babesession := createTestSession(t, nil)
	descriptor := babesession.Descriptor()

	vm := newTestVerificationManager(t, false, descriptor)

	block, _ := createTestBlock(t, babesession, genesisHeader, [][]byte{})
	err := vm.blockState.AddBlock(block)
	require.Nil(t, err)

	ok, err := vm.VerifyBlock(block.Header)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	require.Equal(t, (*types.Header)(nil), vm.firstBlock)
}

func TestVerificationManager_VerifyBlock_WithDigest(t *testing.T) {
	babesession := createTestSession(t, nil)
	descriptor := babesession.Descriptor()

	vm := newTestVerificationManager(t, false, descriptor)
	vm.currentEpoch = 0

	block, _ := createTestBlock(t, babesession, genesisHeader, [][]byte{})

	consensusDigest := &types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              descriptor.Encode(),
	}

	conDigest := consensusDigest.Encode()
	block.Header.Digest = [][]byte{block.Header.Digest[0], conDigest}
	block.Header.Number = big.NewInt(2)

	// re-sign block
	seal, err := babesession.buildBlockSeal(block.Header)
	require.Nil(t, err)

	encSeal := seal.Encode()
	block.Header.Digest = append(block.Header.Digest, encSeal)

	err = vm.blockState.AddBlock(block)
	require.Nil(t, err)

	ok, err := vm.VerifyBlock(block.Header)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	require.Equal(t, block.Header, vm.firstBlock)

	// create block with lower number, check that it's chosen as first block of epoch
	block.Header.Number = big.NewInt(1)

	seal, err = babesession.buildBlockSeal(block.Header)
	require.Nil(t, err)

	encSeal = seal.Encode()
	block.Header.Digest = append(block.Header.Digest, encSeal)

	ok, err = vm.VerifyBlock(block.Header)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	require.Equal(t, block.Header, vm.firstBlock)

	// create block with higher number, check that it's not chosen as first block of epoch
	expected := block.Header.DeepCopy()
	newBlock := &types.Block{
		Header: block.Header.DeepCopy(),
	}

	newBlock.Header.Number = big.NewInt(99)
	seal, err = babesession.buildBlockSeal(newBlock.Header)
	require.Nil(t, err)

	encSeal = seal.Encode()
	newBlock.Header.Digest = append(newBlock.Header.Digest, encSeal)

	ok, err = vm.VerifyBlock(newBlock.Header)
	require.Nil(t, err)
	require.Equal(t, true, ok)
	require.Equal(t, expected, vm.firstBlock)
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

	authorityData := make([]*types.BABEAuthorityData, 1)
	authorityData[0] = &types.BABEAuthorityData{
		ID: kp.Public().(*sr25519.PublicKey),
	}
	babesession.authorityData = authorityData

	verifier, err := newEpochVerifier(babesession.blockState, &NextEpochDescriptor{
		Authorities: babesession.authorityData,
		Randomness:  babesession.config.Randomness,
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
	block, _ := createTestBlock(t, babesession, genesisHeader, [][]byte{})

	verifier, err := newEpochVerifier(babesession.blockState, &NextEpochDescriptor{
		Authorities: babesession.authorityData,
		Randomness:  babesession.config.Randomness,
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

	babesession.authorityData = make([]*types.BABEAuthorityData, 1)
	babesession.authorityData[0] = &types.BABEAuthorityData{
		ID: kp.Public().(*sr25519.PublicKey),
	}

	// create and add first block
	block, _ := createTestBlock(t, babesession, genesisHeader, [][]byte{})
	block.Header.Hash()

	err = babesession.blockState.AddBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	verifier, err := newEpochVerifier(babesession.blockState, &NextEpochDescriptor{
		Authorities: babesession.authorityData,
		Randomness:  babesession.config.Randomness,
	})
	if err != nil {
		t.Fatal(err)
	}

	ok, err := verifier.verifyAuthorshipRight(block.Header)
	require.NoError(t, err)
	require.True(t, ok)

	// create new block
	block2, _ := createTestBlock(t, babesession, genesisHeader, [][]byte{})
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
