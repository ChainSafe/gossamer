// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package modules

import (
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
)

// GrandpaModule init parameters
type GrandpaModule struct {
	blockAPI         BlockAPI
	blockFinalityAPI BlockFinalityAPI
}

// NewGrandpaModule creates a new Grandpa rpc module.
func NewGrandpaModule(api BlockAPI, finalityAPI BlockFinalityAPI) *GrandpaModule {
	return &GrandpaModule{
		blockAPI:         api,
		blockFinalityAPI: finalityAPI,
	}
}

// Votes struct formats rpc call
type Votes struct {
	CurrentWeight uint32   `json:"currentWeight"`
	Missing       []string `json:"missing"`
}

// RoundState json format for roundState RPC call
type RoundState struct {
	Round           uint32 `json:"round"`
	TotalWeight     uint32 `json:"totalWeight"`
	ThresholdWeight uint32 `json:"thresholdWeight"`
	Prevotes        Votes  `json:"prevotes"`
	Precommits      Votes  `json:"precommits"`
}

// RoundStateResponse response to roundState RPC call
type RoundStateResponse struct {
	SetID      uint32       `json:"setId"`
	Best       RoundState   `json:"best"`
	Background []RoundState `json:"background"`
}

// ProveFinalityRequest request struct
type ProveFinalityRequest struct {
	blockHashStart common.Hash
	blockHashEnd   common.Hash
	authorityID    uint64
}

// ProveFinalityResponse is an optional SCALE encoded proof array
type ProveFinalityResponse [][]byte

// ProveFinality for the provided block range. Returns NULL if there are no known finalised blocks in the range. If no authorities set is provided, the current one will be attempted.
func (gm *GrandpaModule) ProveFinality(r *http.Request, req *ProveFinalityRequest, res *ProveFinalityResponse) error {
	blocksToCheck, err := gm.blockAPI.SubChain(req.blockHashStart, req.blockHashEnd)
	if err != nil {
		return err
	}

	// Leaving check in for linter
	if req.authorityID != uint64(0) {
		// TODO: #1404 Check if functionality relevant
	}

	for _, block := range blocksToCheck {
		hasJustification, _ := gm.blockAPI.HasJustification(block)
		if !hasJustification {
			continue
		}

		justification, err := gm.blockAPI.GetJustification(block)
		if err != nil {
			continue
		}
		*res = append(*res, justification)
	}

	return nil
}

// RoundState returns the state of the current best round state as well as the ongoing background rounds.
func (gm *GrandpaModule) RoundState(r *http.Request, req *EmptyRequest, res *RoundStateResponse) error {
	voters := gm.blockFinalityAPI.GetVoters()
	votersPkBytes := make([]ed25519.PublicKeyBytes, len(voters))
	for i, v := range voters {
		votersPkBytes[i] = v.PublicKeyBytes()
	}

	votes := gm.blockFinalityAPI.PreVotes()
	commits := gm.blockFinalityAPI.PreCommits()

	missingPrevotes, err := toAddress(difference(votersPkBytes, votes))
	if err != nil {
		return err
	}

	missingPrecommits, err := toAddress(difference(votersPkBytes, commits))
	if err != nil {
		return err
	}

	totalWeight := uint32(len(voters))
	roundstate := RoundStateResponse{
		SetID: uint32(gm.blockFinalityAPI.GetSetID()),
		Best: RoundState{
			Round:           uint32(gm.blockFinalityAPI.GetRound()),
			ThresholdWeight: thresholdWeight(totalWeight),
			TotalWeight:     totalWeight,
			Prevotes: Votes{
				CurrentWeight: uint32(len(votes)),
				Missing:       missingPrevotes,
			},
			Precommits: Votes{
				CurrentWeight: uint32(len(commits)),
				Missing:       missingPrecommits,
			},
		},
		Background: []RoundState{},
	}

	*res = roundstate
	return nil
}

func thresholdWeight(totalWeight uint32) uint32 {
	return totalWeight * 2 / 3
}

// difference get the values representing the difference, i.e., the values that are in voters but not in pre.
// this function returns the authorities that haven't voted yet
func difference(voters, equivocations []ed25519.PublicKeyBytes) []ed25519.PublicKeyBytes {
	diff := make([]ed25519.PublicKeyBytes, 0)
	diffmap := make(map[ed25519.PublicKeyBytes]bool, len(voters))

	for _, eq := range equivocations {
		diffmap[eq] = true
	}

	for _, v := range voters {
		if _, ok := diffmap[v]; !ok {
			diff = append(diff, v)
		}
	}

	return diff
}

func toAddress(pkb []ed25519.PublicKeyBytes) ([]string, error) {
	addrs := make([]string, len(pkb))
	for i, b := range pkb {
		pk, err := ed25519.NewPublicKey(b[:])
		if err != nil {
			return nil, err
		}

		addrs[i] = string(pk.Address())
	}

	return addrs, nil
}
