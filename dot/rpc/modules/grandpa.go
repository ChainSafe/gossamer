// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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
	BlockNumber uint32 `json:"blockNumber"`
}

// ProveFinalityResponse is an optional SCALE encoded proof array
type ProveFinalityResponse []string

// ProveFinality for the provided block number, returning the Justification for the last block in the set.
func (gm *GrandpaModule) ProveFinality(r *http.Request, req *ProveFinalityRequest, res *ProveFinalityResponse) error {
	blockHash, err := gm.blockAPI.GetHashByNumber(uint(req.BlockNumber))
	if err != nil {
		return err
	}
	hasJustification, err := gm.blockAPI.HasJustification(blockHash)
	if err != nil {
		return err
	}
	
	if !hasJustification {
		*res = append(*res, "GRANDPA prove finality rpc failed: Block not covered by authority set changes")
		return nil
	}
	justification, err := gm.blockAPI.GetJustification(blockHash)
	if err != nil {
		return err
	}
	*res = append(*res, common.BytesToHex(justification))

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
