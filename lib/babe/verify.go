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
)

// ErrNilNextEpochDescriptor is returned when attempting to get a NextEpochDescriptor that isn't set for an epoch
var ErrNilNextEpochDescriptor = errors.New("nil NextEpochDescriptor for epoch")

// VerificationManager assists the syncer in keeping track of what epoch is it currently syncing and verifying,
// as well as keep track of the NextEpochDesciptor which is required to create a Verifier.
type VerificationManager struct {
	epochToNextEpochDescriptor map[uint64]*NextEpochDescriptor
	currentEpoch               uint64
}

func NewVerificationManager(currentEpoch uint64) *VerificationManager {
	return &VerificationManager{
		epochToNextEpochDescriptor: make(map[uint64]*NextEpochDescriptor),
		currentEpoch:               currentEpoch,
	}
}

func (v *VerificationManager) CurrentEpoch() uint64 {
	return v.currentEpoch
}

func (v *VerificationManager) SetCurrentEpoch(epoch uint64) {
	v.currentEpoch = epoch
}

func (v *VerificationManager) IncrementEpoch() {
	v.currentEpoch++
}

func (v *VerificationManager) SetNextEpochDescriptor(epoch uint64, descriptor *NextEpochDescriptor) {
	v.epochToNextEpochDescriptor[epoch] = descriptor
}

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
func (b *Verifier) verifyAuthorshipRight(slot uint64, header *types.Header) (bool, error) {
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
