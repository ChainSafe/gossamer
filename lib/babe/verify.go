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
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// verifierInfo contains the information needed to verify blocks
// it remains the same for an epoch
type verifierInfo struct {
	authorities []*types.Authority
	randomness  [types.RandomnessLength]byte
	threshold   *big.Int
}

// onDisabledInfo contains information about an authority that's been disabled at a certain
// block for the rest of the epoch. the block hash is used to check if the block being verified
// is a descendent of the block that included the `OnDisabled` digest.
type onDisabledInfo struct {
	blockNumber *big.Int
	blockHash   common.Hash
}

// VerificationManager deals with verification that a BABE block producer was authorized to produce a given block.
// It trakcs the BABE epoch data that is needed for verification.
type VerificationManager struct {
	lock       sync.RWMutex
	blockState BlockState
	epochState EpochState
	epochInfo  map[uint64]*verifierInfo // map of epoch number -> info needed for verification
	// there may be different OnDisabled digests on different branches of the chain, so we need to keep track of all of them.
	onDisabled map[uint64]map[uint32][]*onDisabledInfo // map of epoch number -> block producer index -> block number and hash
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
		onDisabled: make(map[uint64]map[uint32][]*onDisabledInfo),
	}, nil
}

// SetOnDisabled sets the BABE authority with the given index as disabled for the rest of the epoch
func (v *VerificationManager) SetOnDisabled(index uint32, header *types.Header) error {
	epoch, err := v.epochState.GetEpochForBlock(header)
	if err != nil {
		return err
	}

	v.lock.Lock()
	defer v.lock.Unlock()

	if _, has := v.epochInfo[epoch]; !has {
		info, err := v.getVerifierInfo(epoch)
		if err != nil {
			return err
		}

		v.epochInfo[epoch] = info
	}

	// check that index is valid
	if index >= uint32(len(v.epochInfo[epoch].authorities)) {
		return ErrInvalidBlockProducerIndex
	}

	// no authorities have been disabled yet this epoch, init map
	if _, has := v.onDisabled[epoch]; !has {
		v.onDisabled[epoch] = make(map[uint32][]*onDisabledInfo)
	}

	disabledProducers := v.onDisabled[epoch]

	if _, has := disabledProducers[index]; !has {
		disabledProducers[index] = []*onDisabledInfo{
			{
				blockNumber: header.Number,
				blockHash:   header.Hash(),
			},
		}
		return nil
	}

	// this producer has already been disabled in this epoch on some branch
	producerInfos := disabledProducers[index]

	// check that the OnDisabled digest isn't a duplicate; ie. that the producer isn't already disabled on this branch
	for _, info := range producerInfos {
		isDescendant, err := v.blockState.IsDescendantOf(info.blockHash, header.Hash())
		if err != nil {
			return err
		}

		if isDescendant && header.Number.Cmp(info.blockNumber) >= 0 {
			// this authority has already been disabled on this branch
			return ErrAuthorityAlreadyDisabled
		}
	}

	disabledProducers[index] = append(producerInfos, &onDisabledInfo{
		blockNumber: header.Number,
		blockHash:   header.Hash(),
	})

	return nil
}

// VerifyBlock verifies that the block producer for the given block was authorized to produce it.
// It returns an error if the block is invalid.
func (v *VerificationManager) VerifyBlock(header *types.Header) error {
	logger.Info("VerifyBlock")
	epoch, err := v.epochState.GetEpochForBlock(header)
	if err != nil {
		logger.Info("cannot GetEpochForBlock ")
		return err
	}

	logger.Info(" GetEpochForBlock ", "epoch", epoch)

	var (
		info *verifierInfo
		has  bool
	)

	v.lock.Lock()

	if info, has = v.epochInfo[epoch]; !has {

		// special case for block 1 - the network doesn't necessarily start in epoch 1.
		// if this happens, the database will be missing info for epochs before the first block.
		if header.Number.Cmp(big.NewInt(1)) == 0 {
			info, err = v.getVerifierInfo(1)
		} else {
			info, err = v.getVerifierInfo(epoch)
		}

		if err != nil {
			v.lock.Unlock()
			return err
		}

		v.epochInfo[epoch] = info
	}

	v.lock.Unlock()

	isDisabled, err := v.isDisabled(epoch, header)
	if err != nil {
		return err
	}

	if isDisabled {
		return ErrAuthorityDisabled
	}

	verifier, err := newVerifier(v.blockState, info)
	if err != nil {
		return err
	}

	return verifier.verifyAuthorshipRight(header)
}

func (v *VerificationManager) isDisabled(epoch uint64, header *types.Header) (bool, error) {
	v.lock.RLock()
	defer v.lock.RUnlock()

	// check if any authorities have been disabled this epoch
	if _, has := v.onDisabled[epoch]; !has {
		return false, nil
	}

	// if authorities have been disabled, check which ones
	idx, err := getAuthorityIndex(header)
	if err != nil {
		return false, err
	}

	if _, has := v.onDisabled[epoch][idx]; !has {
		return false, nil
	}

	// this authority has been disabled on some branch, check if we are on that branch
	producerInfos := v.onDisabled[epoch][idx]
	for _, info := range producerInfos {
		isDescendant, err := v.blockState.IsDescendantOf(info.blockHash, header.Hash())
		if err != nil {
			return false, err
		}

		if isDescendant && header.Number.Cmp(info.blockNumber) > 0 {
			// this authority has been disabled on this branch
			return true, nil
		}
	}

	return false, nil
}

func (v *VerificationManager) getVerifierInfo(epoch uint64) (*verifierInfo, error) {
	epochData, err := v.epochState.GetEpochData(epoch)
	if err != nil {
		return nil, err
	}

	configData, err := v.getConfigData(epoch)
	if err != nil {
		return nil, err
	}

	threshold, err := CalculateThreshold(configData.C1, configData.C2, len(epochData.Authorities))
	if err != nil {
		return nil, err
	}

	return &verifierInfo{
		authorities: epochData.Authorities,
		randomness:  epochData.Randomness,
		threshold:   threshold,
	}, nil
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

// verifyAuthorshipRight verifies that the authority that produced a block was authorized to produce it.
func (b *verifier) verifyAuthorshipRight(header *types.Header) error {
	// header should have 2 digest items (possibly more in the future)
	// first item should be pre-digest, second should be seal
	if len(header.Digest) < 2 {
		return fmt.Errorf("block header is missing digest items")
	}

	logger.Info("verifyAuthorshipRight")

	// check for valid seal by verifying signature
	preDigestItem := header.Digest[0]
	sealItem := header.Digest[len(header.Digest)-1]

	preDigest, ok := preDigestItem.(*types.PreRuntimeDigest)
	if !ok {
		return fmt.Errorf("first digest item is not pre-digest")
	}

	seal, ok := sealItem.(*types.SealDigest)
	if !ok {
		return fmt.Errorf("last digest item is not seal")
	}

	babePreDigest, err := b.verifyPreRuntimeDigest(preDigest)
	if err != nil {
		return fmt.Errorf("failed to verify pre-runtime digest: %w", err)
	}

	logger.Info("verifyAuthorshipRight babePreDigest verified")

	authorPub := b.authorities[babePreDigest.AuthorityIndex()].Key
	logger.Info("verifyAuthorshipRight", "auth key", authorPub.Encode())

	// remove seal before verifying signature
	header.Digest = header.Digest[:len(header.Digest)-1]
	defer func() {
		header.Digest = append(header.Digest, sealItem)
	}()

	encHeader, err := header.Encode()
	if err != nil {
		return err
	}

	// verify the seal is valid
	hash, err := common.Blake2bHash(encHeader)
	if err != nil {
		return err
	}

	ok, err = authorPub.Verify(hash[:], seal.Data)
	if err != nil {
		return err
	}

	if !ok {
		return ErrBadSignature
	}

	// check if the producer has equivocated, ie. have they produced a conflicting block?
	hashes := b.blockState.GetAllBlocksAtDepth(header.ParentHash)

	for _, hash := range hashes {
		currentHeader, err := b.blockState.GetHeader(hash)
		if err != nil {
			continue
		}

		currentBlockProducerIndex, err := getAuthorityIndex(currentHeader)
		if err != nil {
			continue
		}

		existingBlockProducerIndex := babePreDigest.AuthorityIndex()

		if currentBlockProducerIndex == existingBlockProducerIndex && hash != header.Hash() {
			return ErrProducerEquivocated
		}
	}

	return nil
}

func (b *verifier) verifyPreRuntimeDigest(digest *types.PreRuntimeDigest) (types.BabePreRuntimeDigest, error) {
	r := &bytes.Buffer{}
	_, _ = r.Write(digest.Data)
	babePreDigest, err := types.DecodeBabePreDigest(r)
	if err != nil {
		return nil, err
	}

	logger.Info("verifyPreRuntimeDigest", "len(b.authorities)", len(b.authorities), "AuthorityIndex", babePreDigest.AuthorityIndex())

	if len(b.authorities) <= int(babePreDigest.AuthorityIndex()) {
		return nil, ErrInvalidBlockProducerIndex
	}

	var (
		ok bool
	)

	switch d := babePreDigest.(type) {
	case *types.BabePrimaryPreDigest:
		ok, err = b.verifySlotWinner(d.AuthorityIndex(), d.SlotNumber(), d.VrfOutput(), d.VrfProof())
	case *types.BabeSecondaryVRFPreDigest:
		ok, err = b.verifySlotWinner(d.AuthorityIndex(), d.SlotNumber(), d.VrfOutput(), d.VrfProof())
	case *types.BabeSecondaryPlainPreDigest:
		// TODO: implement BABE secondary slot assignment
		logger.Warn("not validating BabeSecondaryPlainPreDigest: BABE secondary slot assignment not implemented")
		return babePreDigest, nil
	}

	// verify that they are the slot winner
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, ErrBadSlotClaim
	}

	return babePreDigest, nil
}

// verifySlotWinner verifies the claim for a slot
func (b *verifier) verifySlotWinner(authorityIndex uint32, slot uint64, vrfOutput [sr25519.VrfOutputLength]byte, vrfProof [sr25519.VrfProofLength]byte) (bool, error) {
	output := big.NewInt(0).SetBytes(vrfOutput[:])
	if output.Cmp(b.threshold) >= 0 {
		return false, ErrVRFOutputOverThreshold
	}

	pub := b.authorities[authorityIndex].Key

	slotBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBytes, slot)
	vrfInput := append(slotBytes, b.randomness[:]...)

	pk, err := sr25519.NewPublicKey(pub.Encode())
	if err != nil {
		return false, err
	}

	return pk.VrfVerify(vrfInput, vrfOutput[:], vrfProof[:])
}

func getAuthorityIndex(header *types.Header) (uint32, error) {
	if len(header.Digest) == 0 {
		return 0, fmt.Errorf("no digest provided")
	}

	digestItem := header.Digest[0]

	preDigest, ok := digestItem.(*types.PreRuntimeDigest)
	if !ok {
		return 0, fmt.Errorf("first digest item is not pre-runtime digest")
	}

	r := &bytes.Buffer{}
	_, _ = r.Write(preDigest.Data)
	babePreDigest, err := types.DecodeBabePreDigest(r)
	if err != nil {
		return 0, fmt.Errorf("cannot decode babe header from pre-digest: %s", err)
	}

	return babePreDigest.AuthorityIndex(), nil
}
