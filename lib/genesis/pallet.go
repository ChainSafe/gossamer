// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// according to chain/westend-local-spec.json
type staking struct {
	ValidatorCount        uint32         `json:"validatorCount"`
	MinimumValidatorCount uint32         `json:"minimumValidatorCount"`
	Invulnerables         []string       `json:"invulnerables"`
	ForceEra              string         `json:"forceEra"`
	SlashRewardFraction   uint32         `json:"slashRewardFraction"`
	CanceledPayout        *scale.Uint128 `json:"canceledPayout"`
	MinNominatorBond      uint32         `json:"minNominatorBond"`
	MinValidatorBond      uint32         `json:"minValidatorBond"`
	MaxValidatorCount     *uint32        `json:"maxValidatorCount"`
	MaxNominatorCount     *uint32        `json:"maxNominatorCount"`
	// TODO: figure out below fields storage key. (#1868)
	// Stakers               [][]interface{} `json:"stakers"`
}

type session struct {
	NextKeys []nextKey `json:"nextKeys"`
}

type nextKey struct {
	AccountID1 string
	AccountID2 string
	KeyOwner   keyOwner
}

type keyOwner struct {
	Grandpa            string `json:"grandpa"`
	Babe               string `json:"babe"`
	ImOnline           string `json:"im_online"`
	ParaValidator      string `json:"para_validator"`
	ParaAssignment     string `json:"para_assignment"`
	AuthorityDiscovery string `json:"authority_discovery"`
}

func (b *nextKey) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&b.AccountID1, &b.AccountID2, &b.KeyOwner}
	return json.Unmarshal(buf, &tmp)

}

func (b nextKey) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{&b.AccountID1, &b.AccountID2, &b.KeyOwner}
	buf, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
