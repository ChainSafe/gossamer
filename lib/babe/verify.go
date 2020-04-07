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
	"fmt"
	log "github.com/ChainSafe/log15"
	"reflect"

	"github.com/ChainSafe/gossamer/dot/core/types"
	babetypes "github.com/ChainSafe/gossamer/lib/babe/types"
)

// verifySlotWinner verifies the claim for a slot, given the BabeHeader for that slot.
func (b *Session) verifySlotWinner(slot uint64, header *babetypes.BabeHeader) (bool, error) {
	if len(b.authorityData) <= int(header.BlockProducerIndex) {
		return false, fmt.Errorf("no authority data for index %d", header.BlockProducerIndex)
	}

	pub := b.authorityData[header.BlockProducerIndex].ID

	slotBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBytes, slot)
	vrfInput := append(slotBytes, b.config.Randomness)

	return pub.VrfVerify(vrfInput, header.VrfOutput[:], header.VrfProof[:])
}

// verifyAuthorshipRight verifies that the authority that produced a block was authorized to produce it.
func (b *Session) verifyAuthorshipRight(slot uint64, header *types.Header) (bool, error) {
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

	// #553 check if the producer has equivocated, ie. have they produced a conflicting block?
	hashes, err := b.blockState.GetAllHashesForParentDepth(header)
	if err != nil {
		return false, err
	}

	for key, v := range hashes {
		//get header
		currentHeader, err := b.blockState.GetHeader(key)
		if err != nil {
			return false, fmt.Errorf("could not verify digests, key %s, depth, %d", key, v)
		}

		currentSeal := currentHeader.Digest[len(currentHeader.Digest)-1][:32]
		existingSeal := header.Digest[len(header.Digest)-1][:32]

		log.Debug("compare SealDigest",
			"\nc.SealDigest", fmt.Sprintf("0x%x", currentSeal),
			"\nh.SealDigest", fmt.Sprintf("0x%x", existingSeal))

		if reflect.DeepEqual(currentSeal, existingSeal) {
			return false, fmt.Errorf("duplicated SealDigest")
		}
	}

	return true, nil
}
