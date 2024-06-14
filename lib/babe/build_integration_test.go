// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package babe

import (
	"bytes"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
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
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, nil)

	parentHash := babeService.blockState.GenesisHash()
	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	testEpochData, err := babeService.initiateEpoch(testEpochIndex)
	require.NoError(t, err)

	slot := getSlot(t, rt, time.Now())
	extrinsic := runtime.NewTestExtrinsic(t, rt, parentHash, parentHash, 0, signature.TestKeyringPairAlice,
		"System.remark", []byte{0xab, 0xcd})
	block := createTestBlockWithSlot(t, babeService, emptyHeader, [][]byte{common.MustHexToBytes(extrinsic)},
		testEpochIndex, testEpochData, slot)

	const expectedSecondExtrinsic = "0x042d000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" //nolint:lll
	expectedBlockHeader := &types.Header{
		ParentHash: emptyHeader.Hash(),
		Number:     1,
	}

	require.Equal(t, expectedBlockHeader.ParentHash, block.Header.ParentHash)
	require.Equal(t, expectedBlockHeader.Number, block.Header.Number)
	require.NotEqual(t, block.Header.StateRoot, emptyHash)
	require.NotEqual(t, block.Header.ExtrinsicsRoot, emptyHash)
	require.Equal(t, 3, len(block.Header.Digest))

	// confirm block body is correct
	extsBytes := types.ExtrinsicsArrayToBytesArray(block.Body)
	require.Equal(t, 2, len(extsBytes))
	// The first extrinsic is based on timestamp so is not consistent, but since the second is based on
	// Parachn0 and Newheads inherents this can be asserted against. This works for now since we don't support real
	// parachain data in these inherents currently, but when we do this will need to be updated
	require.Equal(t, expectedSecondExtrinsic, common.BytesToHex(extsBytes[1]))
}

func TestApplyExtrinsicAfterFirstBlockFinalized(t *testing.T) {
	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, nil)
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
	keyRing, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	genesis, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, nil)

	header := types.NewHeader(genesisHeader.Hash(), common.Hash{}, common.Hash{}, 1, types.NewDigest())
	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(bestBlockHash)
	require.NoError(t, err)

	//initialise block header
	err = rt.InitializeBlock(header)
	require.NoError(t, err)

	// build extrinsic
	rawMeta, err := rt.Metadata()
	require.NoError(t, err)
	var metadataBytes []byte
	err = scale.Unmarshal(rawMeta, &metadataBytes)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = codec.Decode(metadataBytes, meta)
	require.NoError(t, err)

	runtimeVersion, err := rt.Version()
	require.NoError(t, err)

	charlie, err := ctypes.NewMultiAddressFromHexAccountID(
		keyRing.KeyCharlie.Public().Hex())
	require.NoError(t, err)

	call, err := ctypes.NewCall(meta, "Balances.transfer", charlie, ctypes.NewUCompactFromUInt(12345))
	require.NoError(t, err)

	// Create the extrinsic
	extrinsic := ctypes.NewExtrinsic(call)
	genesisHash, err := ctypes.NewHashFromHexString(genesisHeader.Hash().String())
	require.NoError(t, err)

	so := ctypes.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: true},
		GenesisHash:        genesisHash,
		Nonce:              ctypes.NewUCompactFromUInt(uint64(0)),
		SpecVersion:        ctypes.U32(runtimeVersion.SpecVersion),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(runtimeVersion.TransactionVersion),
	}

	// Sign the transaction using Alice's default account
	err = extrinsic.Sign(signature.TestKeyringPairAlice, so)
	require.NoError(t, err)

	extEnc := bytes.NewBuffer(nil)
	encoder := cscale.NewEncoder(extEnc)
	err = extrinsic.Encode(*encoder)
	require.NoError(t, err)

	externalExtrinsic := buildLocalTransaction(t, rt, extEnc.Bytes(), bestBlockHash)

	txVal, err := rt.ValidateTransaction(externalExtrinsic)
	require.NoError(t, err)

	validTransaction := transaction.NewValidTransaction(extEnc.Bytes(), txVal)
	_, err = babeService.transactionState.Push(validTransaction)
	require.NoError(t, err)

	// apply extrinsic
	res, err := rt.ApplyExtrinsic(extEnc.Bytes())
	require.NoError(t, err)
	// Expected result for valid ApplyExtrinsic is 0, 0
	require.Equal(t, []byte{0, 0}, res)
}

func TestBuildAndApplyExtrinsic_InvalidPayment(t *testing.T) {
	keyRing, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	genesis, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, nil)

	header := types.NewHeader(genesisHeader.Hash(), common.Hash{}, common.Hash{}, 1, types.NewDigest())
	bestBlockHash := babeService.blockState.BestBlockHash()
	rt, err := babeService.blockState.GetRuntime(й)
	require.NoError(t, err)

	err = rt.InitializeBlock(header)
	require.NoError(t, err)

	rawMeta, err := rt.Metadata()
	require.NoError(t, err)
	var metadataBytes []byte
	err = scale.Unmarshal(rawMeta, &metadataBytes)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = codec.Decode(metadataBytes, meta)
	require.NoError(t, err)

	runtimeVersion, err := rt.Version()
	require.NoError(t, err)

	charlie, err := ctypes.NewMultiAddressFromHexAccountID(
		keyRing.KeyCharlie.Public().Hex())
	require.NoError(t, err)

	call, err := ctypes.NewCall(meta, "Balances.transfer", charlie, ctypes.NewUCompactFromUInt(^uint64(0)))
	require.NoError(t, err)

	extrinsic := ctypes.NewExtrinsic(call)
	genesisHash, err := ctypes.NewHashFromHexString(genesisHeader.Hash().String())
	require.NoError(t, err)

	so := ctypes.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: true},
		GenesisHash:        genesisHash,
		Nonce:              ctypes.NewUCompactFromUInt(uint64(0)),
		SpecVersion:        ctypes.U32(runtimeVersion.SpecVersion),
		Tip:                ctypes.NewUCompactFromUInt(^uint64(0)),
		TransactionVersion: ctypes.U32(runtimeVersion.TransactionVersion),
	}

	err = extrinsic.Sign(signature.TestKeyringPairAlice, so)
	require.NoError(t, err)

	extEnc := bytes.NewBuffer(nil)
	encoder := cscale.NewEncoder(extEnc)
	err = extrinsic.Encode(*encoder)
	require.NoError(t, err)

	res, err := rt.ApplyExtrinsic(extEnc.Bytes())
	require.NoError(t, err)

	err = DetermineErr(res)
	_, ok := err.(*TransactionValidityError)
	require.True(t, ok)
	require.Equal(t, "transaction validity error: invalid payment", err.Error())
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
	babeService := createTestService(t, ServiceConfig{}, genesis, genesisTrie, genesisHeader, nil)

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
	require.Equal(t, int64(1), timerMetrics.Snapshot().Count())

	// TODO: there isn't an easy way to trigger an error in buildBlock from here
	// _, err = babeService.buildBlock(parent, Slot{}, rt, 0, nil)
	// require.Error(t, err)
	// buildErrorsMetrics := metrics.GetOrRegisterCounter(buildBlockErrors, nil)
	// require.Equal(t, int64(1), buildErrorsMetrics.Count())
}
