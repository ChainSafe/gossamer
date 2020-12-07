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
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// verifierInfo contains the information needed to verify blocks
// it remains the same for an epoch
type verifierInfo struct {
	authorities []*types.Authority
	randomness  [types.RandomnessLength]byte
	threshold   *big.Int
}

// VerificationManager deals with verification that a BABE block producer was authorized to produce a given block.
// It trakcs the BABE epoch data that is needed for verification.
type VerificationManager struct {
	lock       sync.RWMutex
	blockState BlockState
	epochState EpochState
	epochInfo  map[uint64]*verifierInfo // map of epoch number -> info needed for verification
}

// NewVerificationManager returns a new NewVerificationManager
func NewVerificationManager(blockState BlockState, epochState EpochState) (*VerificationManager, error) {
	if blockState == nil {
		return nil, ErrNilBlockState
	}

	if epochState == nil {
		return nil, ErrNilEpochState
	}

	return &VerificationManager{
		epochState: epochState,
		blockState: blockState,
		epochInfo:  make(map[uint64]*verifierInfo),
	}, nil
}

// SetOnDisabled sets the BABE authority with the given index as disabled for the rest of the epoch
func (v *VerificationManager) SetOnDisabled(index uint64, header *types.Header) {
	// TODO: see issue #1205
}

func (v *VerificationManager) VerifyBlock(header *types.Header) (bool, error) {
	epoch, err := v.epochState.GetEpochForBlock(header)
	if err != nil {
		return false, err
	}

	var (
		info *verifierInfo
		has  bool
	)

	v.lock.Lock()

	if info, has = v.epochInfo[epoch]; !has {
		epochData, err := v.epochState.GetEpochData(epoch)
		if err != nil {
			return false, err
		}

		configData, err := v.getConfigData(epoch)
		if err != nil {
			return false, err
		}

		threshold, err := CalculateThreshold(configData.C1, configData.C2, len(epochData.Authorities))
		if err != nil {
			return false, err
		}

		info = &verifierInfo{
			authorities: epochData.Authorities,
			randomness:  epochData.Randomness,
			threshold:   threshold,
		}

		v.epochInfo[epoch] = info
	}

	v.lock.Unlock()

	verifier, err := newVerifier(v.blockState, info)
	if err != nil {
		return false, err
	}

	return verifier.verifyAuthorshipRight(header)
}

func (v *VerificationManager) getConfigData(epoch uint64) (*types.ConfigData, error) {
	for i := epoch; i > 0; i-- {
		has, err := v.epochState.HasConfigData(i)
		if err != nil {
			return nil, err
		}

		if has {
			return v.epochState.GetConfigData(i)
		}
	}

	return nil, errors.New("cannot find ConfigData for epoch")
}

// verifier is a BABE verifier for a specific authority set, randomness, and threshold
type verifier struct {
	blockState  BlockState
	authorities []*types.Authority
	randomness  [types.RandomnessLength]byte
	threshold   *big.Int
}

// newVerifier returns a Verifier for the epoch described by the given descriptor
func newVerifier(blockState BlockState, info *verifierInfo) (*verifier, error) {
	if blockState == nil {
		return nil, ErrNilBlockState
	}

	return &verifier{
		blockState:  blockState,
		authorities: info.authorities,
		randomness:  info.randomness,
		threshold:   info.threshold,
	}, nil
}

// verifySlotWinner verifies the claim for a slot, given the BabeHeader for that slot.
func (b *verifier) verifySlotWinner(slot uint64, header *types.BabeHeader) (bool, error) {
	if len(b.authorities) <= int(header.BlockProducerIndex) {
		return false, ErrInvalidBlockProducerIndex
	}

	// check that vrf output is under threshold
	// if not, then return an error
	output := big.NewInt(0).SetBytes(header.VrfOutput[:])
	if output.Cmp(b.threshold) >= 0 {
		return false, ErrVRFOutputOverThreshold
	}

	pub := b.authorities[header.BlockProducerIndex].Key

	slotBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBytes, slot)
	vrfInput := append(slotBytes, b.randomness[:]...)

	sr25519PK, err := sr25519.NewPublicKey(pub.Encode())
	if err != nil {
		return false, err
	}

	return sr25519PK.VrfVerify(vrfInput, header.VrfOutput[:], header.VrfProof[:])
}

// verifyAuthorshipRight verifies that the authority that produced a block was authorized to produce it.
func (b *verifier) verifyAuthorshipRight(header *types.Header) (bool, error) {
	// header should have 2 digest items (possibly more in the future)
	// first item should be pre-digest, second should be seal
	if len(header.Digest) < 2 {
		return false, fmt.Errorf("block header is missing digest items")
	}

	// check for valid seal by verifying signature
	preDigestBytes := header.Digest[0]
	sealBytes := header.Digest[len(header.Digest)-1]

	digestItem, err := types.DecodeDigestItem(preDigestBytes)
	if err != nil {
		return false, err
	}

	preDigest, ok := digestItem.(*types.PreRuntimeDigest)
	if !ok {
		return false, fmt.Errorf("first digest item is not pre-digest")
	}

	digestItem, err = types.DecodeDigestItem(sealBytes)
	if err != nil {
		return false, err
	}

	seal, ok := digestItem.(*types.SealDigest)
	if !ok {
		return false, fmt.Errorf("last digest item is not seal")
	}

	babeHeader := new(types.BabeHeader)
	err = babeHeader.Decode(preDigest.Data)
	if err != nil {
		return false, fmt.Errorf("cannot decode babe header from pre-digest: %s", err)
	}

	if len(b.authorities) <= int(babeHeader.BlockProducerIndex) {
		return false, ErrInvalidBlockProducerIndex
	}

	slot := babeHeader.SlotNumber

	authorPub := b.authorities[babeHeader.BlockProducerIndex].Key
	// remove seal before verifying
	header.Digest = header.Digest[:len(header.Digest)-1]
	encHeader, err := header.Encode()
	if err != nil {
		return false, err
	}

	// verify that they are the slot winner
	ok, err = b.verifySlotWinner(slot, babeHeader)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, ErrBadSlotClaim
	}

	// verify the seal is valid
	ok, err = authorPub.Verify(encHeader, seal.Data)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, ErrBadSignature
	}

	// check if the producer has equivocated, ie. have they produced a conflicting block?
	hashes := b.blockState.GetAllBlocksAtDepth(header.ParentHash)

	for _, hash := range hashes {
		currentHeader, err := b.blockState.GetHeader(hash)
		if err != nil {
			continue
		}

		currentBlockProducerIndex, err := getBlockProducerIndex(currentHeader)
		if err != nil {
			continue
		}

		existingBlockProducerIndex := babeHeader.BlockProducerIndex

		if currentBlockProducerIndex == existingBlockProducerIndex && hash != header.Hash() {
			return false, ErrProducerEquivocated
		}
	}

	return true, nil
}

func getBlockProducerIndex(header *types.Header) (uint64, error) {
	if len(header.Digest) == 0 {
		return 0, fmt.Errorf("no digest provided")
	}

	preDigestBytes := header.Digest[0]

	digestItem, err := types.DecodeDigestItem(preDigestBytes)
	if err != nil {
		return 0, err
	}

	preDigest, ok := digestItem.(*types.PreRuntimeDigest)
	if !ok {
		return 0, err
	}

	babeHeader := new(types.BabeHeader)
	err = babeHeader.Decode(preDigest.Data)
	if err != nil {
		return 0, err
	}

	return babeHeader.BlockProducerIndex, nil
}
