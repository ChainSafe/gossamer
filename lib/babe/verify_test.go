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
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core/types"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

func TestVerifySlotWinner(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &SessionConfig{
		Keypair: kp,
	}

	babesession := createTestSession(t, cfg)
	err = babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	// create proof that we can authorize this block
	babesession.epochThreshold = big.NewInt(0)
	babesession.authorityIndex = 0
	var slotNumber uint64 = 1

	outAndProof, err := babesession.runLottery(slotNumber)
	if err != nil {
		t.Fatal(err)
	}

	if outAndProof == nil {
		t.Fatal("proof was nil when over threshold")
	}

	babesession.slotToProof[slotNumber] = outAndProof

	slot := Slot{
		start:    uint64(time.Now().Unix()),
		duration: uint64(10000000),
		number:   slotNumber,
	}

	// create babe header
	babeHeader, err := babesession.buildBlockBabeHeader(slot)
	if err != nil {
		t.Fatal(err)
	}

	babesession.authorityData = make([]*AuthorityData, 1)
	babesession.authorityData[0] = &AuthorityData{
		ID: kp.Public().(*sr25519.PublicKey),
	}

	ok, err := babesession.verifySlotWinner(slot.number, babeHeader)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("did not verify slot winner")
	}
}

func TestVerifyAuthorshipRight(t *testing.T) {
	testsCases := []struct {
		description                     string
		parentHeader                    *types.Header
		expectedErr                     error
		authorshipRight                 bool
		expectedErrAfterAuthorshipRight error
	}{
		{
			description:                     "test verify block with existing parent",
			parentHeader:                    genesisHeader,
			expectedErr:                     nil,
			authorshipRight:                 true,
			expectedErrAfterAuthorshipRight: errors.New("duplicated digest"),
		},
		//{
		//	description:     "test verify block with not existing parent",
		//	parentHeader:    nil,
		//	expectedErr:     errors.New("cannot find parent block in blocktree"),
		//	authorshipRight: false,
		//},
	}

	for _, test := range testsCases {
		t.Run(test.description, func(t *testing.T) {

			kp, err := sr25519.GenerateKeypair()
			if err != nil {
				t.Fatal(err)
			}

			cfg := &SessionConfig{
				Keypair: kp,
			}

			babesession := createTestSession(t, cfg)
			err = babesession.configurationFromRuntime()
			if err != nil {
				t.Fatal(err)
			}

			// see https://github.com/noot/substrate/blob/add-blob/core/test-runtime/src/system.rs#L468
			txb := []byte{3, 16, 110, 111, 111, 116, 1, 64, 103, 111, 115, 115, 97, 109, 101, 114, 95, 105, 115, 95, 99, 111, 111, 108}

			genesisHashStr := genesisHeader.Hash().String()
			log.Warn("TestVerifyAuthorshipRight, ", "genesisHashStr", genesisHashStr)

			block, slot := createTestBlock(babesession, [][]byte{txb}, t, test.parentHeader)

			ok, err := babesession.verifyAuthorshipRight(slot.number, block.Header)
			require.Equal(t, test.expectedErr, err)

			require.Equal(t, test.authorshipRight, ok, "did not verify authorship right")

			if test.authorshipRight {
				//save block
				arrivalTime := uint64(time.Now().Unix()) - slot.duration

				err = babesession.blockState.AddBlockWithArrivalTime(block, arrivalTime)
				if err != nil {
					t.Fatal(err)
				}

				// create proof that we can authorize this block
				babesession.epochThreshold = big.NewInt(2)
				babesession.authorityIndex = 2
				var slotNumber uint64 = 2

				outAndProof, err := babesession.runLottery(slotNumber)
				if err != nil {
					t.Fatal(err)
				}

				if outAndProof == nil {
					t.Fatal("proof was nil when over threshold")
				}

				babesession.slotToProof[slotNumber] = outAndProof

				slotNew := Slot{
					start:    uint64(time.Now().Unix()),
					duration: uint64(10000000),
					number:   slotNumber,
				}

				//create new block
				blockNew, _ := createTestBlock(babesession, [][]byte{txb}, t, test.parentHeader)

				// create babe header
				babeHeader, err := babesession.buildBlockBabeHeader(slot)
				if err != nil {
					t.Fatal(err)
				}

				babesession.authorityData = make([]*AuthorityData, 1)
				babesession.authorityData[0] = &AuthorityData{
					ID: kp.Public().(*sr25519.PublicKey),
				}

				ok, err := babesession.verifySlotWinner(slotNew.number, babeHeader)
				if err != nil {
					t.Fatal(err)
				}

				if !ok {
					t.Fatal("did not verify slot winner")
				}

				//update blockNumber to previous block
				//blockNew.Header.Number = block.Header.Number

				//verifyAuthorshipRight
				ok, err = babesession.verifyAuthorshipRight(slotNew.number, blockNew.Header)
				require.False(t, ok)

				require.NotNil(t, err)
				require.Equal(t, test.expectedErrAfterAuthorshipRight, err)

			}
		})
	}
}
