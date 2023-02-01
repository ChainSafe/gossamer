// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"bytes"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"

	cscale "github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/stretchr/testify/require"
)

func TestSeal(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	builder := &BlockBuilder{
		keypair: kp,
	}

	zeroHash, err := common.HexToHash("0x00")
	require.NoError(t, err)

	header := types.NewHeader(zeroHash, zeroHash, zeroHash, 0, types.NewDigest())

	encHeader, err := scale.Marshal(*header)
	require.NoError(t, err)

	hash, err := common.Blake2bHash(encHeader)
	require.NoError(t, err)

	seal, err := builder.buildBlockSeal(header)
	require.NoError(t, err)

	ok, err := kp.Public().Verify(hash[:], seal.Data)
	require.NoError(t, err)

	require.True(t, ok, "could not verify seal")
}

// TODO see if there can be better assertions on block body #3060
// Are extrinsics correct, what are the extrinsics now that there are 2 instead of 1, is one the same?
// Does order matter?
func TestBuildBlock_ok(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader)

	parentHash := babeService.blockState.GenesisHash()
	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	slot := getSlot(t, rt, time.Now())
	ext := runtime.NewTestExtrinsic(t, rt, parentHash, parentHash, 0, signature.TestKeyringPairAlice,
		"System.remark", []byte{0xab, 0xcd})
	block := createTestBlockWithSlot(t, babeService, emptyHeader, [][]byte{common.MustHexToBytes(ext)},
		testEpochIndex, epochData, slot)

	expectedBlockHeader := &types.Header{
		ParentHash: emptyHeader.Hash(),
		Number:     1,
	}

	require.Equal(t, expectedBlockHeader.ParentHash, block.Header.ParentHash)
	require.Equal(t, expectedBlockHeader.Number, block.Header.Number)
	require.NotEqual(t, block.Header.StateRoot, emptyHash)
	require.NotEqual(t, block.Header.ExtrinsicsRoot, emptyHash)
	require.Equal(t, 3, len(block.Header.Digest.Types))

	// confirm block body is correct
	extsBytes := types.ExtrinsicsArrayToBytesArray(block.Body)
	require.Equal(t, 2, len(extsBytes))
}

func TestApplyExtrinsicAfterFirstBlockFinalized(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader)
	const authorityIndex = 0

	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	slot := getSlot(t, rt, time.Now())
	preRuntimeDigest, err := claimSlot(testEpochIndex, slot.number, epochData, babeService.keypair)
	require.NoError(t, err)

	builder := NewBlockBuilder(
		babeService.keypair,
		babeService.transactionState,
		babeService.blockState,
		authorityIndex,
		preRuntimeDigest,
	)

	parentHeader := emptyHeader
	number := parentHeader.Number + 1
	digest := types.NewDigest()
	err = digest.Add(*builder.preRuntimeDigest)
	require.NoError(t, err)
	header := types.NewHeader(parentHeader.Hash(), common.Hash{}, common.Hash{}, number, digest)

	err = rt.InitializeBlock(header)
	require.NoError(t, err)

	_, err = buildBlockInherents(slot, rt, parentHeader)
	require.NoError(t, err)

	ext := runtime.NewTestExtrinsic(t, rt, emptyHash, parentHeader.Hash(), 0, signature.TestKeyringPairAlice,
		"System.remark", []byte{0xab, 0xcd})
	_, err = rt.ApplyExtrinsic(common.MustHexToBytes(ext))
	require.NoError(t, err)

	header1, err := rt.FinalizeBlock()
	require.NoError(t, err)

	ext2 := runtime.NewTestExtrinsic(t, rt, parentHeader.Hash(), parentHeader.Hash(), 0,
		signature.TestKeyringPairAlice, "System.remark",
		[]byte{0xab, 0xcd})

	validExt := []byte{byte(types.TxnExternal)}
	validExt = append(validExt, common.MustHexToBytes(ext2)...)
	validExt = append(validExt, babeService.blockState.BestBlockHash().ToBytes()...)
	_, err = rt.ValidateTransaction(validExt)
	require.NoError(t, err)

	// Add 7 seconds to allow slot to be claimed at appropriate time, Westend has 6 second slot times
	slot2 := getSlot(t, rt, time.Now().Add(7*time.Second))
	preRuntimeDigest2, err := claimSlot(testEpochIndex, slot2.number, epochData, babeService.keypair)
	require.NoError(t, err)

	digest2 := types.NewDigest()
	err = digest2.Add(*preRuntimeDigest2)
	require.NoError(t, err)
	header2 := types.NewHeader(header1.Hash(), common.Hash{}, common.Hash{}, 2, digest2)
	err = rt.InitializeBlock(header2)
	require.NoError(t, err)

	_, err = buildBlockInherents(slot2, rt, header1)
	require.NoError(t, err)

	res, err := rt.ApplyExtrinsic(common.MustHexToBytes(ext2))
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0}, res)

	_, err = rt.FinalizeBlock()
	require.NoError(t, err)
}

func TestBuildAndApplyExtrinsic(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader)

	parentHash := common.MustHexToHash("0x35a28a7dbaf0ba07d1485b0f3da7757e3880509edc8c31d0850cb6dd6219361d")
	header := types.NewHeader(parentHash, common.Hash{}, common.Hash{}, 1, types.NewDigest())

	bestBlockHash := babeService.blockState.BestBlockHash()
	runtime, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	//initialise block header
	err = runtime.InitializeBlock(header)
	require.NoError(t, err)

	// build extrinsic
	rawMeta, err := runtime.Metadata()
	require.NoError(t, err)
	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = codec.Decode(decoded, meta)
	require.NoError(t, err)

	rv := runtime.Version()

	bob, err := ctypes.NewMultiAddressFromHexAccountID(
		"0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22")
	require.NoError(t, err)

	call, err := ctypes.NewCall(meta, "Balances.transfer", bob, ctypes.NewUCompactFromUInt(12345))
	require.NoError(t, err)

	// Create the extrinsic
	ext := ctypes.NewExtrinsic(call)
	genHash, err := ctypes.NewHashFromHexString("0x35a28a7dbaf0ba07d1485b0f3da7757e3880509edc8c31d0850cb6dd6219361d")
	require.NoError(t, err)

	o := ctypes.SignatureOptions{
		BlockHash:          genHash,
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: true},
		GenesisHash:        genHash,
		Nonce:              ctypes.NewUCompactFromUInt(uint64(0)),
		SpecVersion:        ctypes.U32(rv.SpecVersion),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion),
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	extEnc := bytes.Buffer{}
	encoder := cscale.NewEncoder(&extEnc)
	ext.Encode(*encoder)

	externalExtrinsic := buildLocalTransaction(t, runtime, extEnc.Bytes(), bestBlockHash)

	txVal, err := runtime.ValidateTransaction(externalExtrinsic)
	require.NoError(t, err)

	vtx := transaction.NewValidTransaction(extEnc.Bytes(), txVal)
	babeService.transactionState.Push(vtx)

	// apply extrinsic
	res, err := runtime.ApplyExtrinsic(extEnc.Bytes())
	require.NoError(t, err)
	// Expected result for valid ApplyExtrinsic is 0, 0
	require.Equal(t, []byte{0, 0}, res)
}

// TODO investigate if this is a needed test #3060
// Good to test build block error case, but this test can be improved
func TestBuildBlock_failing(t *testing.T) {
	t.Skip()

	gen, genTrie, genHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, gen, genTrie, genHeader)

	// see https://github.com/noot/substrate/blob/add-blob/core/test-runtime/src/system.rs#L468
	// add a valid transaction
	txa := []byte{
		3, 16, 110, 111, 111, 116,
		1, 64, 103, 111, 115, 115,
		97, 109, 101, 114, 95, 105,
		115, 95, 99, 111, 111, 108}
	vtx := transaction.NewValidTransaction(types.Extrinsic(txa), &transaction.Validity{})
	babeService.transactionState.Push(vtx)

	// add a transaction that can't be included (transfer from account with no balance)
	// See https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	txb := []byte{
		1, 212, 53, 147, 199, 21, 253, 211,
		28, 97, 20, 26, 189, 4, 169, 159,
		214, 130, 44, 133, 88, 133, 76, 205,
		227, 154, 86, 132, 231, 165, 109, 162,
		125, 142, 175, 4, 21, 22, 135, 115, 99,
		38, 201, 254, 161, 126, 37, 252, 82,
		135, 97, 54, 147, 201, 18, 144, 156,
		178, 38, 170, 71, 148, 242, 106, 72,
		69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 216, 5, 113, 87, 87, 40,
		221, 120, 247, 252, 137, 201, 74, 231,
		222, 101, 85, 108, 102, 39, 31, 190, 210,
		14, 215, 124, 19, 160, 180, 203, 54,
		110, 167, 163, 149, 45, 12, 108, 80,
		221, 65, 238, 57, 237, 199, 16, 10,
		33, 185, 8, 244, 184, 243, 139, 5,
		87, 252, 245, 24, 225, 37, 154, 163, 142}
	vtx = transaction.NewValidTransaction(types.Extrinsic(txb), &transaction.Validity{})
	babeService.transactionState.Push(vtx)

	zeroHash, err := common.HexToHash("0x00")
	require.NoError(t, err)

	parentHeader := &types.Header{
		ParentHash: zeroHash,
	}

	duration, err := time.ParseDuration("1s")
	require.NoError(t, err)

	slot := Slot{
		start:    time.Now(),
		duration: duration,
		number:   1000,
	}

	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	const authorityIndex uint32 = 0
	_, err = babeService.buildBlock(parentHeader, slot, rt, authorityIndex, &types.PreRuntimeDigest{})
	require.NotNil(t, err)
	require.Equal(t, "cannot build extrinsics: error applying extrinsic: Apply error, type: Payment",
		err.Error(), "Did not receive expected error text")

	txc := babeService.transactionState.(*state.TransactionState).Peek()
	if !bytes.Equal(txc.Extrinsic, txa) {
		t.Fatal("did not readd valid transaction to queue")
	}
}

func TestDecodeExtrinsicBody(t *testing.T) {
	ext := types.NewExtrinsic([]byte{0x1, 0x2, 0x3})
	inh := [][]byte{{0x4, 0x5}, {0x6, 0x7}}

	vtx := transaction.NewValidTransaction(ext, &transaction.Validity{})

	body, err := extrinsicsToBody(inh, []*transaction.ValidTransaction{vtx})
	require.NoError(t, err)
	require.NotNil(t, body)
	require.Len(t, body, 3)

	contains, err := body.HasExtrinsic(ext)
	require.NoError(t, err)
	require.True(t, contains)
}

func TestBuildBlockTimeMonitor(t *testing.T) {
	metrics.Enabled = true
	metrics.Unregister(buildBlockTimer)

	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader)

	parent, err := babeService.blockState.BestBlockHeader()
	require.NoError(t, err)

	runtime, err := babeService.blockState.GetRuntime(parent.Hash())
	require.NoError(t, err)

	timerMetrics := metrics.GetOrRegisterTimer(buildBlockTimer, nil)
	timerMetrics.Stop()

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	slot := getSlot(t, runtime, time.Now())
	createTestBlockWithSlot(t, babeService, parent, [][]byte{}, testEpochIndex, epochData, slot)
	require.Equal(t, int64(1), timerMetrics.Count())

	// TODO: there isn't an easy way to trigger an error in buildBlock from here
	// _, err = babeService.buildBlock(parent, Slot{}, rt, 0, nil)
	// require.Error(t, err)
	// buildErrorsMetrics := metrics.GetOrRegisterCounter(buildBlockErrors, nil)
	// require.Equal(t, int64(1), buildErrorsMetrics.Count())
}
