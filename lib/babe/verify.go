// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var errEmptyKeyOwnershipProof = errors.New("key ownership proof is nil")

// verifierInfo contains the information needed to verify blocks
// it remains the same for an epoch
type verifierInfo struct {
	authorities    []types.AuthorityRaw
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
// It tracks the BABE epoch data that is needed for verification.
type VerificationManager struct {
	lock       sync.Mutex
	blockState BlockState
	slotState  SlotState
	epochState EpochState
	epochInfo  map[uint64]*verifierInfo // map of epoch number -> info needed for verification
	// there may be different OnDisabled digests on different
	// branches of the chain, so we need to keep track of all of them.
	// map of epoch number -> block producer index -> block number and hash
	onDisabled map[uint64]map[uint32][]*onDisabledInfo
}

// NewVerificationManager returns a new NewVerificationManager
func NewVerificationManager(blockState BlockState, slotState SlotState, epochState EpochState) *VerificationManager {
	return &VerificationManager{
		epochState: epochState,
		slotState:  slotState,
		blockState: blockState,
		epochInfo:  make(map[uint64]*verifierInfo),
		onDisabled: make(map[uint64]map[uint32][]*onDisabledInfo),
	}
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
	parentHeader, err := v.blockState.GetHeader(header.ParentHash)
	if err != nil {
		return fmt.Errorf("getting header: %w", err)
	}

	currentBlockEpoch, err := v.epochState.GetEpochForBlock(header)
	if err != nil {
		return fmt.Errorf("getting epoch for block header: %w", err)
	}

	epochWhereDataDescriptorIs := currentBlockEpoch
	if parentHeader.Hash() != v.blockState.GenesisHash() {
		parentEpoch, err := v.epochState.GetEpochForBlock(parentHeader)
		if err != nil {
			return fmt.Errorf("getting epoch for parent header: %w", err)
		}

		if parentEpoch > currentBlockEpoch {
			return fmt.Errorf("%w: expected epoch greater than parent block epoch %d, got: %d",
				errEpochLowerThanExpected, parentEpoch, currentBlockEpoch)
		}

		if currentBlockEpoch > (parentEpoch + 1) {
			epochWhereDataDescriptorIs = parentEpoch + 1
		}
	}

	slotDuration, err := v.epochState.GetSlotDuration()
	if err != nil {
		return fmt.Errorf("getting current slot duration: %w", err)
	}

	info, err := v.getVerifierInfo(epochWhereDataDescriptorIs, header)
	if err != nil {
		return fmt.Errorf("getting verifier info: %w", err)
	}

	verifier := newVerifier(v.blockState, v.slotState, currentBlockEpoch, info, slotDuration)
	return verifier.verifyAuthorshipRight(header)
}

func (v *VerificationManager) getVerifierInfo(epoch uint64, header *types.Header) (*verifierInfo, error) {
	epochData, err := v.epochState.GetEpochDataRaw(epoch, header)
	if err != nil {
		return nil, fmt.Errorf("failed to get epoch data for epoch %d: %w", epoch, err)
	}

	configData, err := v.epochState.GetConfigData(epoch, header)
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

// verifier is a BABE verifier for a specific authority set, randomness, and threshold
type verifier struct {
	blockState     BlockState
	slotState      SlotState
	epoch          uint64
	authorities    []types.AuthorityRaw
	randomness     Randomness
	threshold      *scale.Uint128
	secondarySlots bool
	slotDuration   time.Duration
}

// newVerifier returns a Verifier for the epoch described by the given descriptor
func newVerifier(blockState BlockState, slotState SlotState,
	epoch uint64, info *verifierInfo, slotDuration time.Duration) *verifier {
	return &verifier{
		blockState:     blockState,
		slotState:      slotState,
		epoch:          epoch,
		authorities:    info.authorities,
		randomness:     info.randomness,
		threshold:      info.threshold,
		secondarySlots: info.secondarySlots,
		slotDuration:   slotDuration,
	}
}

// verifyAuthorshipRight verifies that the authority that produced a block was authorized to produce it.
func (b *verifier) verifyAuthorshipRight(header *types.Header) error {
	// header should have 2 digest items (possibly more in the future)
	// first item should be pre-digest, second should be seal
	if len(header.Digest) < 2 {
		return errMissingDigestItems
	}

	logger.Tracef("beginning BABE authorship right verification for block %s", header.Hash())

	// check for valid seal by verifying signature
	preDigestItem := header.Digest[0]
	sealItem := header.Digest[len(header.Digest)-1]

	preDigestItemValue, err := preDigestItem.Value()
	if err != nil {
		return fmt.Errorf("getting pre digest item value: %w", err)
	}
	preDigest, ok := preDigestItemValue.(types.PreRuntimeDigest)
	if !ok {
		return fmt.Errorf("%w: got %T", types.ErrNoFirstPreDigest, preDigestItemValue)
	}

	sealItemValue, err := sealItem.Value()
	if err != nil {
		return fmt.Errorf("getting seal item value: %w", err)
	}
	seal, ok := sealItemValue.(types.SealDigest)
	if !ok {
		return fmt.Errorf("%w: got %T", errLastDigestItemNotSeal, sealItemValue)
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

	authority := new(types.Authority)
	err = authority.FromRawSr25519(&b.authorities[authIdx])
	if err != nil {
		return fmt.Errorf("from raw sr25519: %w", err)
	}

	// remove seal before verifying signature
	h := types.NewDigest()
	for _, val := range header.Digest[:len(header.Digest)-1] {
		digestValue, err := val.Value()
		if err != nil {
			return fmt.Errorf("getting digest type value: %w", err)
		}
		err = h.Add(digestValue)
		if err != nil {
			return err
		}
	}

	header.Digest = h
	defer func() {
		sealItemVal, err := sealItem.Value()
		if err != nil {
			logger.Errorf("getting seal item value: %s", err)
		}
		if err = header.Digest.Add(sealItemVal); err != nil {
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

	ok, err = authority.Key.Verify(hash[:], seal.Data)
	if err != nil {
		return err
	}

	if !ok {
		return ErrBadSignature
	}

	equivocated, err := b.verifyBlockEquivocation(header)
	if err != nil {
		return fmt.Errorf("could not verify block equivocation: %w", err)
	}

	if equivocated {
		return fmt.Errorf("%w for block header %s", ErrProducerEquivocated, header.Hash())
	}

	return nil
}

func (b *verifier) submitAndReportEquivocation(equivocationProof *types.BabeEquivocationProof) error {
	bestBlockHash := b.blockState.BestBlockHash()
	runtimeInstance, err := b.blockState.GetRuntime(bestBlockHash)
	if err != nil {
		return fmt.Errorf("getting runtime: %w", err)
	}

	keyOwnershipProof, err := runtimeInstance.BabeGenerateKeyOwnershipProof(
		equivocationProof.Slot, equivocationProof.Offender)
	if err != nil {
		return fmt.Errorf("getting key ownership proof from runtime: %w", err)
	} else if keyOwnershipProof == nil {
		return errEmptyKeyOwnershipProof
	}

	err = runtimeInstance.BabeSubmitReportEquivocationUnsignedExtrinsic(*equivocationProof, keyOwnershipProof)
	if err != nil {
		return fmt.Errorf("submitting equivocation report to runtime: %w", err)
	}

	return nil
}

// verifyBlockEquivocation checks if the given block's author has occupied the corresponding slot more than once.
// It returns true if the block was equivocated.
// TODO: Check if it is initial sync
// don't report any equivocations during initial sync
// as they are most likely stale.
// https://github.com/ChainSafe/gossamer/issues/3004
func (b *verifier) verifyBlockEquivocation(header *types.Header) (bool, error) {
	authorityIndex, slotNumber, err := getAuthorityIndexAndSlot(header)
	if err != nil {
		return false, fmt.Errorf("failed to get authority index: %w", err)
	}

	if len(b.authorities) <= int(authorityIndex) {
		return false, ErrAuthIndexOutOfBound
	}

	if header.Hash() == b.blockState.GenesisHash() {
		return false, nil
	}

	slotNow := getCurrentSlot(b.slotDuration)
	signer := b.authorities[authorityIndex].Key
	equivocationProof, err := b.slotState.CheckEquivocation(slotNow, slotNumber,
		header, signer)
	if err != nil {
		return false, fmt.Errorf("checking equivocation: %w", err)
	}

	// WHy are we not in this case??
	if equivocationProof == nil {
		return false, nil
	}

	err = b.submitAndReportEquivocation(equivocationProof)
	if err != nil {
		return false, fmt.Errorf("submiting equivocation: %w", err)
	}

	return true, nil
}

func (b *verifier) verifyPreRuntimeDigest(digest *types.PreRuntimeDigest) (any, error) {
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

	if uint64(len(b.authorities)) <= uint64(authIdx) {
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
		authority := b.authorities[d.AuthorityIndex]
		pk, err := sr25519.NewPublicKey(authority.Key[:])
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
	authority := b.authorities[authorityIndex]
	pk, err := sr25519.NewPublicKey(authority.Key[:])
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
		"public key 0x%x, randomness 0x%x, slot %d, epoch %d, "+
		"output 0x%x and proof 0x%x",
		authorityIndex,
		authority.Key[:], b.randomness, slot, b.epoch,
		vrfOutput, vrfProof[:])

	t := makeTranscript(b.randomness, slot, b.epoch)
	return pk.VrfVerify(t, vrfOutput, vrfProof)
}

func getAuthorityIndexAndSlot(header *types.Header) (authIdx uint32, slot uint64, err error) {
	if len(header.Digest) == 0 {
		return 0, 0, fmt.Errorf("for block hash %s: %w", header.Hash(), errNoDigest)
	}

	digestValue, err := header.Digest[0].Value()
	if err != nil {
		return 0, 0, fmt.Errorf("getting first digest type value: %w", err)
	}
	preDigest, ok := digestValue.(types.PreRuntimeDigest)
	if !ok {
		return 0, 0, types.ErrNoFirstPreDigest
	}

	babePreDigest, err := types.DecodeBabePreDigest(preDigest.Data)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot decode babe header from pre-digest: %s", err)
	}

	switch d := babePreDigest.(type) {
	case types.BabePrimaryPreDigest:
		authIdx = d.AuthorityIndex
		slot = d.SlotNumber
	case types.BabeSecondaryVRFPreDigest:
		authIdx = d.AuthorityIndex
		slot = d.SlotNumber
	case types.BabeSecondaryPlainPreDigest:
		authIdx = d.AuthorityIndex
		slot = d.SlotNumber
	}

	return authIdx, slot, nil
}
