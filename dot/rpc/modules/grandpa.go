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
	"github.com/ChainSafe/gossamer/lib/grandpa"
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

type Votes struct {
	CurrentWeight uint32   `json:"currentWeight"`
	Missing       []string `json:"missing"`
}

type RoundState struct {
	Round           uint32 `json:"round"`
	TotalWeight     uint32 `json:"totalWeight"`
	ThresholdWeight uint32 `json:"thresholdWeight"`
	Prevotes        Votes  `json:"prevotes"`
	Precommits      Votes  `json:"precommits"`
}

type RoundStateResponse struct {
	SetId      uint32       `json:"setId"`
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
	// if req.authorityID != uint64(0) {
	// 	// TODO: #1404 Check if functionality relevant
	// }

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

	prevotes := gm.blockFinalityAPI.PreVotes()
	precommits := gm.blockFinalityAPI.PreCommits()

	missingPrevotes, err := toAddress(difference(votersPkBytes, prevotes))
	if err != nil {
		return err
	}

	missingPrecommits, err := toAddress(difference(votersPkBytes, precommits))
	if err != nil {
		return err
	}

	totalWeight := calcTotalWeight(voters)

	roundstate := RoundStateResponse{
		SetId: uint32(gm.blockFinalityAPI.GetSetID()),
		Best: RoundState{
			Round:           uint32(gm.blockFinalityAPI.GetRound()),
			ThresholdWeight: calcThresholdWeight(totalWeight),
			TotalWeight:     totalWeight,
			Prevotes: Votes{
				CurrentWeight: calcWeight(voters, prevotes),
				Missing:       missingPrevotes,
			},
			Precommits: Votes{
				CurrentWeight: calcWeight(voters, precommits),
				Missing:       missingPrecommits,
			},
		},
		Background: []RoundState{},
	}

	*res = roundstate
	return nil
}

func calcWeight(voters grandpa.Voters, pre map[ed25519.PublicKeyBytes]*grandpa.Vote) uint32 {
	var weight uint32
	for pk := range pre {
		for _, gpv := range voters {
			if gpv.PublicKeyBytes() == pk {
				weight += uint32(gpv.ID)
			}
		}
	}
	return weight
}

func calcTotalWeight(voters grandpa.Voters) uint32 {
	var totalWeight uint32
	for _, v := range voters {
		totalWeight += uint32(v.ID)
	}

	return totalWeight
}

func calcThresholdWeight(totalWeight uint32) uint32 {
	faulty := (totalWeight - 1) / 3
	return totalWeight - faulty
}

// difference get the values representing the difference, i.e., the values that are in voters but not in pre.
func difference(voters []ed25519.PublicKeyBytes, pre map[ed25519.PublicKeyBytes]*grandpa.Vote) []ed25519.PublicKeyBytes {
	diff := make([]ed25519.PublicKeyBytes, 0)

	for _, v := range voters {
		if _, ok := pre[v]; !ok {
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
