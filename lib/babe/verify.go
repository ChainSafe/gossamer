// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// verifierInfo contains the information needed to verify blocks
// it remains the same for an epoch
type verifierInfo struct {
	authorities    []types.Authority
	randomness     Randomness
	threshold      *scale.Uint128
	secondarySlots bool
}

// onDisabledInfo contains information about an authority that's been disabled at a certain
// block for the rest of the epoch. the block hash is used to check if the block being verified
// is a descendent of the block that included the `OnDisabled` digest.
type onDisabledInfo struct {
	blockNumber uint
	blockHash   common.Hash
}

// VerificationManager deals with verification that a BABE block producer was authorized to produce a given block.
// It trakcs the BABE epoch data that is needed for verification.
type VerificationManager struct {
	lock       sync.RWMutex
	blockState BlockState
	epochState EpochState
	epochInfo  map[uint64]*verifierInfo // map of epoch number -> info needed for verification
	// there may be different OnDisabled digests on different
	// branches of the chain, so we need to keep track of all of them.
	// map of epoch number -> block producer index -> block number and hash
	onDisabled map[uint64]map[uint32][]*onDisabledInfo
}

// NewVerificationManager returns a new NewVerificationManager
func NewVerificationManager(blockState BlockState, epochState EpochState) (*VerificationManager, error) {
	if blockState == nil {
		return nil, ErrNilBlockState
	}

	if epochState == nil {
		return nil, errNilEpochState
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
		info, err := v.getVerifierInfo(epoch, header)
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

		if isDescendant && header.Number >= info.blockNumber {
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
// It checks the next epoch and config data stored in memory only if it cannot retrieve the data from database
// It returns an error if the block is invalid.
func (v *VerificationManager) VerifyBlock(header *types.Header) error {
	var (
		info *verifierInfo
		has  bool
	)

	// special case for block 1 - the network doesn't necessarily start in epoch 1.
	// if this happens, the database will be missing info for epochs before the first block.
	if header.Number == 1 {
		block1IsFinal, err := v.blockState.NumberIsFinalised(header.Number)
		if err != nil {
			return fmt.Errorf("failed to check if block 1 is finalised: %w", err)
		}

		if !block1IsFinal {
			firstSlot, err := types.GetSlotFromHeader(header)
			if err != nil {
				return fmt.Errorf("failed to get slot from block 1: %w", err)
			}

			logger.Debugf("syncing block 1, setting first slot as %d", firstSlot)

			err = v.epochState.SetFirstSlot(firstSlot)
			if err != nil {
				return fmt.Errorf("failed to set current epoch after receiving block 1: %w", err)
			}
		}
	}

	epoch, err := v.epochState.GetEpochForBlock(header)
	if err != nil {
		return fmt.Errorf("failed to get epoch for block header: %w", err)
	}

	v.lock.Lock()

	if info, has = v.epochInfo[epoch]; !has {
		info, err = v.getVerifierInfo(epoch, header)
		if err != nil {
			v.lock.Unlock()
			// SkipVerify is set to true only in the case where we have imported a state at a given height,
			// thus missing the epoch data for previous epochs.
			skip, skipErr := v.epochState.SkipVerify(header)
			if skipErr != nil {
				return fmt.Errorf("failed to check if verification can be skipped: %w", skipErr)
			}

			if skip {
				return nil
			}

			return fmt.Errorf("failed to get verifier info for block %d: %w", header.Number, err)
		}

		v.epochInfo[epoch] = info
	}

	v.lock.Unlock()

	verifier, err := newVerifier(v.blockState, epoch, info)
	if err != nil {
		return fmt.Errorf("failed to create new BABE verifier: %w", err)
	}

	return verifier.verifyAuthorshipRight(header)
}

func (v *VerificationManager) getVerifierInfo(epoch uint64, header *types.Header) (*verifierInfo, error) {
	epochData, err := v.epochState.GetEpochData(epoch, header)
	if err != nil {
		return nil, fmt.Errorf("failed to get epoch data for epoch %d: %w", epoch, err)
	}

	configData, err := v.getConfigData(epoch, header)
	if err != nil {
		return nil, fmt.Errorf("failed to get config data: %w", err)
	}

	threshold, err := CalculateThreshold(configData.C1, configData.C2, len(epochData.Authorities))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate threshold: %w", err)
	}

	return &verifierInfo{
		authorities:    epochData.Authorities,
		randomness:     epochData.Randomness,
		threshold:      threshold,
		secondarySlots: configData.SecondarySlots > 0,
	}, nil
}

func (v *VerificationManager) getConfigData(epoch uint64, header *types.Header) (*types.ConfigData, error) {
	for i := int(epoch); i >= 0; i-- {
		has, err := v.epochState.HasConfigData(uint64(i))
		if err != nil {
			return nil, err
		} else if !has {
			continue
		}

		return v.epochState.GetConfigData(uint64(i), header)
	}

	return nil, errNoConfigData
}

// verifier is a BABE verifier for a specific authority set, randomness, and threshold
type verifier struct {
	blockState     BlockState
	epoch          uint64
	authorities    []types.Authority
	randomness     Randomness
	threshold      *scale.Uint128
	secondarySlots bool
}

// newVerifier returns a Verifier for the epoch described by the given descriptor
func newVerifier(blockState BlockState, epoch uint64, info *verifierInfo) (*verifier, error) {
	if blockState == nil {
		return nil, ErrNilBlockState
	}

	return &verifier{
		blockState:     blockState,
		epoch:          epoch,
		authorities:    info.authorities,
		randomness:     info.randomness,
		threshold:      info.threshold,
		secondarySlots: info.secondarySlots,
	}, nil
}

// verifyAuthorshipRight verifies that the authority that produced a block was authorized to produce it.
func (b *verifier) verifyAuthorshipRight(header *types.Header) error {
	// header should have 2 digest items (possibly more in the future)
	// first item should be pre-digest, second should be seal
	if len(header.Digest.Types) < 2 {
		return errMissingDigestItems
	}

	logger.Tracef("beginning BABE authorship right verification for block %s", header.Hash())

	// check for valid seal by verifying signature
	preDigestItem := header.Digest.Types[0]
	sealItem := header.Digest.Types[len(header.Digest.Types)-1]

	preDigest, ok := preDigestItem.Value().(types.PreRuntimeDigest)
	if !ok {
		return fmt.Errorf("first digest item is not pre-digest")
	}

	seal, ok := sealItem.Value().(types.SealDigest)
	if !ok {
		return fmt.Errorf("last digest item is not seal")
	}

	babePreDigest, err := b.verifyPreRuntimeDigest(&preDigest)
	if err != nil {
		return fmt.Errorf("failed to verify pre-runtime digest: %w", err)
	}

	logger.Tracef("verified block %s BABE pre-runtime digest", header.Hash())

	var authIdx uint32
	switch d := babePreDigest.(type) {
	case types.BabePrimaryPreDigest:
		authIdx = d.AuthorityIndex
	case types.BabeSecondaryVRFPreDigest:
		authIdx = d.AuthorityIndex
	case types.BabeSecondaryPlainPreDigest:
		authIdx = d.AuthorityIndex
	}

	authorPub := b.authorities[authIdx].Key

	// remove seal before verifying signature
	h := types.NewDigest()
	for _, val := range header.Digest.Types[:len(header.Digest.Types)-1] {
		err = h.Add(val.Value())
		if err != nil {
			return err
		}
	}

	header.Digest = h
	defer func() {
		if err = header.Digest.Add(sealItem.Value()); err != nil {
			logger.Errorf("failed to re-add seal to digest: %s", err)
		}
	}()

	encHeader, err := scale.Marshal(*header)
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

		var existingBlockProducerIndex uint32
		switch d := babePreDigest.(type) {
		case types.BabePrimaryPreDigest:
			existingBlockProducerIndex = d.AuthorityIndex
		case types.BabeSecondaryVRFPreDigest:
			existingBlockProducerIndex = d.AuthorityIndex
		case types.BabeSecondaryPlainPreDigest:
			existingBlockProducerIndex = d.AuthorityIndex
		}

		if currentBlockProducerIndex == existingBlockProducerIndex && hash != header.Hash() {
			return ErrProducerEquivocated
		}
	}

	return nil
}

func (b *verifier) verifyPreRuntimeDigest(digest *types.PreRuntimeDigest) (scale.VaryingDataTypeValue, error) {
	babePreDigest, err := types.DecodeBabePreDigest(digest.Data)
	if err != nil {
		return nil, err
	}

	var authIdx uint32
	switch d := babePreDigest.(type) {
	case types.BabePrimaryPreDigest:
		authIdx = d.AuthorityIndex
	case types.BabeSecondaryVRFPreDigest:
		authIdx = d.AuthorityIndex
	case types.BabeSecondaryPlainPreDigest:
		authIdx = d.AuthorityIndex
	}

	if int(authIdx) < 0 || int(authIdx) >= len(b.authorities) {
		logger.Tracef("verifyPreRuntimeDigest invalid auth index %d, we have %d auths",
			authIdx, len(b.authorities))
		return nil, ErrInvalidBlockProducerIndex
	}

	var (
		ok bool
	)

	switch d := babePreDigest.(type) {
	case types.BabePrimaryPreDigest:
		ok, err = b.verifyPrimarySlotWinner(d.AuthorityIndex, d.SlotNumber, d.VRFOutput, d.VRFProof)
	case types.BabeSecondaryVRFPreDigest:
		if !b.secondarySlots {
			ok = false
			break
		}
		pub := b.authorities[d.AuthorityIndex].Key

		pk, err := sr25519.NewPublicKey(pub.Encode())
		if err != nil {
			return nil, err
		}

		ok, err = verifySecondarySlotVRF(&d, pk, b.epoch, len(b.authorities), b.randomness)
		if err != nil {
			return nil, err
		}

	case types.BabeSecondaryPlainPreDigest:
		if !b.secondarySlots {
			ok = false
			break
		}

		ok = true
		err = verifySecondarySlotPlain(d.AuthorityIndex, d.SlotNumber, len(b.authorities), b.randomness)
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

// verifyPrimarySlotWinner verifies the claim for a slot
func (b *verifier) verifyPrimarySlotWinner(authorityIndex uint32,
	slot uint64, vrfOutput [sr25519.VRFOutputLength]byte,
	vrfProof [sr25519.VRFProofLength]byte) (bool, error) {
	pub := b.authorities[authorityIndex].Key

	pk, err := sr25519.NewPublicKey(pub.Encode())
	if err != nil {
		return false, err
	}

	// check that VRF output was under threshold
	ok, err := checkPrimaryThreshold(b.randomness,
		slot,
		b.epoch,
		vrfOutput,
		b.threshold,
		pk,
	)
	if err != nil {
		return false, fmt.Errorf("failed to compare with threshold, %w", err)
	}
	if !ok {
		return false, ErrVRFOutputOverThreshold
	}

	// validate VRF proof
	logger.Tracef("verifyPrimarySlotWinner authority index %d, "+
		"public key %s, randomness 0x%x, slot %d, epoch %d, "+
		"output 0x%x and proof 0x%x",
		authorityIndex,
		pub.Hex(), b.randomness, slot, b.epoch,
		vrfOutput, vrfProof[:])

	t := makeTranscript(b.randomness, slot, b.epoch)
	return pk.VrfVerify(t, vrfOutput, vrfProof)
}

func getAuthorityIndex(header *types.Header) (uint32, error) {
	if len(header.Digest.Types) == 0 {
		return 0, fmt.Errorf("no digest provided")
	}

	preDigest, ok := header.Digest.Types[0].Value().(types.PreRuntimeDigest)
	if !ok {
		return 0, fmt.Errorf("first digest item is not pre-runtime digest")
	}

	babePreDigest, err := types.DecodeBabePreDigest(preDigest.Data)
	if err != nil {
		return 0, fmt.Errorf("cannot decode babe header from pre-digest: %s", err)
	}

	var authIdx uint32
	switch d := babePreDigest.(type) {
	case types.BabePrimaryPreDigest:
		authIdx = d.AuthorityIndex
	case types.BabeSecondaryVRFPreDigest:
		authIdx = d.AuthorityIndex
	case types.BabeSecondaryPlainPreDigest:
		authIdx = d.AuthorityIndex
	}

	return authIdx, nil
}
