// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"bytes"
	"fmt"
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

func TestBuildBlock_ok(t *testing.T) {
	gen, genTrie, genHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, gen, genTrie, genHeader)

	parentHash := babeService.blockState.GenesisHash()
	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	slot := getSlot(t, rt, time.Now())
	ext := runtime.NewTestExtrinsic(t, rt, parentHash, parentHash, 0, signature.TestKeyringPairAlice, "System.remark", []byte{0xab, 0xcd})
	block := createTestBlockWithSlot(t, babeService, emptyHeader, [][]byte{common.MustHexToBytes(ext)}, testEpochIndex, epochData, slot)

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
	fmt.Println(extsBytes)
	require.Equal(t, 1, len(extsBytes))
}

func TestApplyExtrinsic(t *testing.T) {
	gen, genTrie, genHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, gen, genTrie, genHeader)
	const authorityIndex = 0

	duration, err := time.ParseDuration("1s")
	require.NoError(t, err)

	slotnum := uint64(1)
	slot := Slot{
		start:    time.Now(),
		duration: duration,
		number:   slotnum,
	}

	slot2 := Slot{
		start:    time.Now(),
		duration: duration,
		number:   2,
	}
	testVRFOutputAndProof := &VrfOutputAndProof{}

	preDigest2, err := types.NewBabePrimaryPreDigest(
		authorityIndex, slot2.number,
		testVRFOutputAndProof.output,
		testVRFOutputAndProof.proof,
	).ToPreRuntimeDigest()
	require.NoError(t, err)

	parentHash := babeService.blockState.GenesisHash()
	parentHeader, err := babeService.blockState.GetHeader(parentHash)
	require.NoError(t, err)

	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	ts, err := babeService.storageState.TrieState(nil)
	require.NoError(t, err)
	rt.SetContextStorage(ts)

	preDigest, err := types.NewBabePrimaryPreDigest(
		authorityIndex, slot.number,
		testVRFOutputAndProof.output,
		testVRFOutputAndProof.proof,
	).ToPreRuntimeDigest()
	require.NoError(t, err)

	digest := types.NewDigest()
	err = digest.Add(*preDigest)
	require.NoError(t, err)

	header := types.NewHeader(parentHash, common.Hash{}, common.Hash{}, 1, digest)

	//initialise block header
	err = rt.InitializeBlock(header)
	require.NoError(t, err)

	_, err = buildBlockInherents(slot, rt, parentHeader)
	require.NoError(t, err)

	header1, err := rt.FinalizeBlock()
	require.NoError(t, err)

	extHex := runtime.NewTestExtrinsic(t, rt, parentHash, parentHash, 0,
		signature.TestKeyringPairAlice, "System.remark", []byte{0xab, 0xcd})
	extBytes := common.MustHexToBytes(extHex)
	_, err = rt.ValidateTransaction(append([]byte{byte(types.TxnExternal)}, extBytes...))
	require.NoError(t, err)

	digest2 := types.NewDigest()
	err = digest2.Add(*preDigest2)
	require.NoError(t, err)
	header2 := types.NewHeader(header1.Hash(), common.Hash{}, common.Hash{}, 2, digest2)
	err = rt.InitializeBlock(header2)
	require.NoError(t, err)

	_, err = buildBlockInherents(slot, rt, parentHeader)
	require.NoError(t, err)

	res, err := rt.ApplyExtrinsic(extBytes)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0}, res)

	_, err = rt.FinalizeBlock()
	require.NoError(t, err)
}

func buildLocalTransaction(t *testing.T, rt runtime.Instance, ext types.Extrinsic,
	bestBlockHash common.Hash) types.Extrinsic {
	runtimeVersion := rt.Version()
	txQueueVersion, err := runtimeVersion.TaggedTransactionQueueVersion()
	require.NoError(t, err)
	var extrinsicParts [][]byte
	switch txQueueVersion {
	case 3:
		extrinsicParts = [][]byte{{byte(types.TxnLocal)}, ext, bestBlockHash.ToBytes()}
	case 2:
		extrinsicParts = [][]byte{{byte(types.TxnLocal)}, ext}
	}
	return types.Extrinsic(bytes.Join(extrinsicParts, nil))
}

func TestBuildAndApplyExtrinsic(t *testing.T) {
	gen, genTrie, genHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, gen, genTrie, genHeader)

	parentHash := common.MustHexToHash("0x35a28a7dbaf0ba07d1485b0f3da7757e3880509edc8c31d0850cb6dd6219361d")
	header := types.NewHeader(parentHash, common.Hash{}, common.Hash{}, 1, types.NewDigest())

	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	//initialise block header
	err = rt.InitializeBlock(header)
	require.NoError(t, err)

	// build extrinsic
	rawMeta, err := rt.Metadata()
	require.NoError(t, err)
	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = codec.Decode(decoded, meta)
	require.NoError(t, err)

	rv := rt.Version()

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

	externalExtrinsic := buildLocalTransaction(t, rt, extEnc.Bytes(), bestBlockHash)

	txVal, err := rt.ValidateTransaction(externalExtrinsic)
	require.NoError(t, err)

	vtx := transaction.NewValidTransaction(extEnc.Bytes(), txVal)
	babeService.transactionState.Push(vtx)

	// apply extrinsic
	res, err := rt.ApplyExtrinsic(extEnc.Bytes())
	require.NoError(t, err)
	// Expected result for valid ApplyExtrinsic is 0, 0
	require.Equal(t, []byte{0, 0}, res)
}

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

	cfg := ServiceConfig{
		Authority: true,
	}

	gen, genTrie, genHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, cfg, gen, genTrie, genHeader)

	parent, err := babeService.blockState.BestBlockHeader()
	require.NoError(t, err)

	timerMetrics := metrics.GetOrRegisterTimer(buildBlockTimer, nil)
	timerMetrics.Stop()

	epochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	createTestBlock(t, babeService, parent, [][]byte{}, 1, testEpochIndex, epochData)
	require.Equal(t, int64(1), timerMetrics.Count())

	// TODO: there isn't an easy way to trigger an error in buildBlock from here
	// _, err = babeService.buildBlock(parent, Slot{}, rt, 0, nil)
	// require.Error(t, err)
	// buildErrorsMetrics := metrics.GetOrRegisterCounter(buildBlockErrors, nil)
	// require.Equal(t, int64(1), buildErrorsMetrics.Count())
}
