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

	"github.com/ChainSafe/gossamer/dot/core/types"
	babetypes "github.com/ChainSafe/gossamer/lib/babe/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// ErrNilNextEpochDescriptor is returned when attempting to get a NextEpochDescriptor that isn't set for an epoch
var ErrNilNextEpochDescriptor = errors.New("nil NextEpochDescriptor for epoch")

// VerificationManager assists the syncer in keeping track of what epoch is it currently syncing and verifying,
// as well as keep track of the NextEpochDesciptor which is required to create a Verifier.
type VerificationManager struct {
	epochToNextEpochDescriptor map[uint64]*NextEpochDescriptor
	blockState                 BlockState
	// TODO: map of epochs to epoch length changes, for use in determining block epoch

	// current epoch information
	currentEpoch uint64
	firstBlock   *types.Header // first block of current epoch, may change over course of epoch
	verifier     *Verifier
}

func NewVerificationManager(currentEpoch uint64) *VerificationManager {
	return &VerificationManager{
		epochToNextEpochDescriptor: make(map[uint64]*NextEpochDescriptor),
		currentEpoch:               currentEpoch,
		verifier:                   &Verifier{},
	}
}

func (v *VerificationManager) CurrentEpoch() uint64 {
	return v.currentEpoch
}

func (v *VerificationManager) SetCurrentEpoch(epoch uint64) {
	v.currentEpoch = epoch
}

func (v *VerificationManager) IncrementEpoch() error {
	if v.firstBlock != nil {
		consensusDigest, err := checkForConsensusDigest(v.firstBlock)
		if err != nil {
			return err
		}

		if consensusDigest == nil {
			return errors.New("first block for next epoch doesn't have consensus digest")
		}

		nextEpochDescriptor := new(NextEpochDescriptor)
		err = nextEpochDescriptor.Decode(consensusDigest.Data)
		if err != nil {
			return err
		}

		v.epochToNextEpochDescriptor[v.currentEpoch] = nextEpochDescriptor
		v.verifier = NewVerifier(nextEpochDescriptor.Authorities, nextEpochDescriptor.Randomness[0])
	}

	v.firstBlock = nil
	v.currentEpoch++
	return nil
}

// func (v *VerificationManager) SetNextEpochDescriptor(epoch uint64, descriptor *NextEpochDescriptor) {
// 	v.epochToNextEpochDescriptor[epoch] = descriptor
// }

func (v *VerificationManager) GetNextEpochDescriptor(epoch uint64) *NextEpochDescriptor {
	return v.epochToNextEpochDescriptor[epoch]
}

// Verifier returns a Verifier for the given epoch.
func (v *VerificationManager) Verifier(epoch uint64) (*Verifier, error) {
	descriptor := v.epochToNextEpochDescriptor[epoch]
	if descriptor == nil {
		return nil, ErrNilNextEpochDescriptor
	}

	return NewVerifier(descriptor.Authorities, descriptor.Randomness[0]), nil
}

func (v *VerificationManager) VerifyBlock(header *types.Header) (bool, error) {
	fromEpoch, err := v.isBlockFromCurrentEpoch(header.Hash())
	if err != nil {
		return false, err
	}

	digest, err := checkForConsensusDigest(header)
	if err != nil {
		return false, err
	}

	ok, err := v.verifier.verifyAuthorshipRight(header)
	if err != nil {
		return false, err
	}

	if digest == nil {
		// verify and return
		return ok, nil
	}

	// check if first block has been set for current epoch
	if fromEpoch && v.firstBlock != nil {

		// check if block header has lower block number than current first block
		if header.Number.Cmp(v.firstBlock.Number) < 0 {
			v.firstBlock = header
		}

	} else if fromEpoch {
		// set first block in current epoch
		v.firstBlock = header
	}

	return ok, nil
}

func checkForConsensusDigest(header *types.Header) (*types.ConsensusDigest, error) {
	// check if block header digest items exist
	if header.Digest == nil || len(header.Digest) == 0 {
		return nil, fmt.Errorf("header digest is not set")
	}

	// declare digest item
	var consensusDigest *types.ConsensusDigest

	// decode each digest item and check its type
	for _, digest := range header.Digest {
		item, err := types.DecodeDigestItem(digest)
		if err != nil {
			return nil, err
		}

		// check if digest item is consensus digest type
		if item.Type() == types.ConsensusDigestType {
			var ok bool
			consensusDigest, ok = item.(*types.ConsensusDigest)
			if ok {
				break
			}
		}
	}

	return consensusDigest, nil
}

// blockFromCurrentEpoch verifies the provided block hash is from current epoch
func (v *VerificationManager) isBlockFromCurrentEpoch(hash common.Hash) (bool, error) {
	// get epoch number of block header
	epoch, err := v.getBlockEpoch(hash)
	if err != nil {
		return false, fmt.Errorf("[babe verifier] failed to get epoch from block header: %s", err)
	}

	// check if block epoch number matches current epoch number
	if epoch != v.currentEpoch {
		return false, nil
	}

	return true, nil
}

// getBlockEpoch gets the epoch number using the provided block hash
func (v *VerificationManager) getBlockEpoch(hash common.Hash) (epoch uint64, err error) {

	// get slot number to determine epoch number
	// TODO: directly calulcate this from the pre-digest
	slot, err := v.blockState.GetSlotForBlock(hash)
	if err != nil {
		return epoch, fmt.Errorf("failed to get slot from block hash: %s", err)
	}

	if slot != 0 {
		// epoch number = (slot - genesis slot) / epoch length
		epoch = (slot - 1) / 6 // TODO: use epoch length from babe or core config
	}

	return epoch, nil
}

// Verifier represents a BABE verifier for a specific epoch
type Verifier struct {
	authorityData []*AuthorityData
	randomness    byte // TODO: update to [32]byte when runtime is updated
}

func NewVerifier(authorityData []*AuthorityData, randomness byte) *Verifier {
	return &Verifier{
		authorityData: authorityData,
		randomness:    randomness, // TODO: update to [32]byte when runtime is updated
	}
}

// verifySlotWinner verifies the claim for a slot, given the BabeHeader for that slot.
func (b *Verifier) verifySlotWinner(slot uint64, header *babetypes.BabeHeader) (bool, error) {
	if len(b.authorityData) <= int(header.BlockProducerIndex) {
		return false, fmt.Errorf("no authority data for index %d", header.BlockProducerIndex)
	}

	pub := b.authorityData[header.BlockProducerIndex].ID

	slotBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBytes, slot)
	vrfInput := append(slotBytes, b.randomness)

	return pub.VrfVerify(vrfInput, header.VrfOutput[:], header.VrfProof[:])
}

// verifyAuthorshipRight verifies that the authority that produced a block was authorized to produce it.
func (b *Verifier) verifyAuthorshipRight(header *types.Header) (bool, error) {
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

	babeHeader := new(babetypes.BabeHeader)
	err = babeHeader.Decode(preDigest.Data)
	if err != nil {
		return false, fmt.Errorf("cannot decode babe header from pre-digest: %s", err)
	}

	if len(b.authorityData) <= int(babeHeader.BlockProducerIndex) {
		return false, fmt.Errorf("no authority data for index %d", babeHeader.BlockProducerIndex)
	}

	slot := babeHeader.SlotNumber

	authorPub := b.authorityData[babeHeader.BlockProducerIndex].ID
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
		return false, fmt.Errorf("could not verify slot claim")
	}

	// verify the seal is valid
	ok, err = authorPub.Verify(encHeader, seal.Data)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, fmt.Errorf("could not verify signature")
	}

	// TODO: check if the producer has equivocated, ie. have they produced a conflicting block?
	return true, nil
}
