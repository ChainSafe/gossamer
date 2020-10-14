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
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/transaction"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func TestSeal(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	cfg := &ServiceConfig{
		Keypair: kp,
	}

	babeService := createTestService(t, cfg)

	zeroHash, err := common.HexToHash("0x00")
	require.NoError(t, err)

	header, err := types.NewHeader(zeroHash, big.NewInt(0), zeroHash, zeroHash, [][]byte{})
	require.NoError(t, err)

	encHeader, err := header.Encode()
	require.NoError(t, err)

	seal, err := babeService.buildBlockSeal(header)
	require.NoError(t, err)

	ok, err := kp.Public().Verify(encHeader, seal.Data)
	require.NoError(t, err)

	require.True(t, ok, "could not verify seal")
}

func addAuthorshipProof(t *testing.T, babeService *Service, slotNumber uint64) {
	outAndProof, err := babeService.runLottery(slotNumber)
	require.NoError(t, err)
	require.NotNil(t, outAndProof, "proof was nil when under threshold")
	babeService.slotToProof[slotNumber] = outAndProof
}

func createTestBlock(t *testing.T, babeService *Service, parent *types.Header, exts [][]byte, slotNumber uint64) (*types.Block, Slot) {
	// create proof that we can authorize this block
	babeService.threshold = maxThreshold
	babeService.authorityIndex = 0

	addAuthorshipProof(t, babeService, slotNumber)

	for _, ext := range exts {
		vtx := transaction.NewValidTransaction(ext, &transaction.Validity{})
		_, _ = babeService.transactionState.Push(vtx)
	}

	slot := Slot{
		start:    uint64(time.Now().Unix()),
		duration: uint64(10000000),
		number:   slotNumber,
	}

	// build block
	var block *types.Block
	var err error

	for i := 0; i < 1; i++ { // retry if error
		block, err = babeService.buildBlock(parent, slot)
		if err == nil {
			return block, slot
		}
	}

	require.NoError(t, err)
	return block, slot
}

func TestBuildBlock_ok(t *testing.T) {
	cfg := &ServiceConfig{
		TransactionState: state.NewTransactionState(),
		LogLvl:           log.LvlDebug,
	}

	babeService := createTestService(t, cfg)

	// TODO: re-add extrinsic
	exts := [][]byte{}

	block, slot := createTestBlock(t, babeService, emptyHeader, exts, 1)

	// create pre-digest
	preDigest, err := babeService.buildBlockPreDigest(slot)
	require.NoError(t, err)

	pdEnc, err := preDigest.Encode()
	require.NoError(t, err)

	expectedBlockHeader := &types.Header{
		ParentHash: emptyHeader.Hash(),
		Number:     big.NewInt(1),
		StateRoot:  emptyHash,
		Digest:     [][]byte{pdEnc},
	}

	// remove seal from built block, since we can't predict the signature
	block.Header.Digest = block.Header.Digest[:1]
	header, err := babeService.blockState.BestBlockHeader()
	require.NoError(t, err)

	// reset StateRoot and ExtrinsicRoot, since it has randomness aspects in it
	// TODO: where does this randomness come from?
	block.Header.ExtrinsicsRoot = header.ExtrinsicsRoot
	block.Header.StateRoot = header.StateRoot
	require.Equal(t, block.Header, expectedBlockHeader)

	// confirm block body is correct
	extsRes, err := block.Body.AsExtrinsics()
	require.NoError(t, err)

	extsBytes := types.ExtrinsicsArrayToBytesArray(extsRes)
	require.Equal(t, exts, extsBytes)
}

func TestBuildBlock_failing(t *testing.T) {
	t.Skip()
	cfg := &ServiceConfig{
		TransactionState: state.NewTransactionState(),
	}

	var err error
	babeService := createTestService(t, cfg)

	babeService.authorityData = []*types.Authority{
		{Key: nil, Weight: 1},
	}

	// create proof that we can authorize this block
	babeService.threshold = big.NewInt(0)
	var slotNumber uint64 = 1

	outAndProof, err := babeService.runLottery(slotNumber)
	require.NoError(t, err)
	require.NotNil(t, outAndProof, "proof was nil when over threshold")

	babeService.slotToProof[slotNumber] = outAndProof

	// see https://github.com/noot/substrate/blob/add-blob/core/test-runtime/src/system.rs#L468
	// add a valid transaction
	txa := []byte{3, 16, 110, 111, 111, 116, 1, 64, 103, 111, 115, 115, 97, 109, 101, 114, 95, 105, 115, 95, 99, 111, 111, 108}
	vtx := transaction.NewValidTransaction(types.Extrinsic(txa), &transaction.Validity{})
	babeService.transactionState.Push(vtx)

	// add a transaction that can't be included (transfer from account with no balance)
	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	txb := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 142}
	vtx = transaction.NewValidTransaction(types.Extrinsic(txb), &transaction.Validity{})
	babeService.transactionState.Push(vtx)

	zeroHash, err := common.HexToHash("0x00")
	require.NoError(t, err)

	parentHeader := &types.Header{
		ParentHash: zeroHash,
		Number:     big.NewInt(0),
	}

	slot := Slot{
		start:    uint64(time.Now().Unix()),
		duration: uint64(10000000),
		number:   slotNumber,
	}

	_, err = babeService.buildBlock(parentHeader, slot)
	if err == nil {
		t.Fatal("should error when attempting to include invalid tx")
	}
	require.Equal(t, "cannot build extrinsics: error applying extrinsic: Apply error, type: Payment",
		err.Error(), "Did not receive expected error text")

	txc := babeService.transactionState.Peek()
	if !bytes.Equal(txc.Extrinsic, txa) {
		t.Fatal("did not readd valid transaction to queue")
	}
}
