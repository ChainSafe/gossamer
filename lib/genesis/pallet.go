// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

type staking struct {
	HistoryDepth          uint32         `json:"historyDepth"`
	ValidatorCount        uint32         `json:"validatorCount"`
	MinimumValidatorCount uint32         `json:"minimumValidatorCount"`
	Invulnerables         []string       `json:"invulnerables"`
	ForceEra              string         `json:"forceEra"`
	SlashRewardFraction   uint32         `json:"slashRewardFraction"`
	CanceledSlashPayout   *scale.Uint128 `json:"canceledSlashPayout"`
	// TODO: figure out below fields storage key. (#1868)
	// Stakers               [][]interface{} `json:"stakers"`
}

// // according to chain/westend-local-spec.json
// type staking struct {
// 	ValidatorCount        uint32         `json:"validatorCount"`
// 	MinimumValidatorCount uint32         `json:"minimumValidatorCount"`
// 	Invulnerables         []string       `json:"invulnerables"`
// 	ForceEra              string         `json:"forceEra"`
// 	SlashRewardFraction   uint32         `json:"slashRewardFraction"`
// 	CanceledPayout        *scale.Uint128 `json:"canceledPayout"`
// 	// Stakers               [][]interface{} `json:"stakers"`
// 	MinNominatorBond  uint32  `json:"minNominatorBond"`
// 	MinValidatorBond  uint32  `json:"minValidatorBond"`
// 	MaxValidatorCount *uint32 `json:"maxValidatorCount"`
// 	MaxNominatorCount *uint32 `json:"maxNominatorCount"`
// }

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
	AuthorityDiscovery string `json:"authority_discovery"`
}

type membersFields struct {
	AccountID string
	Balance   float64
}

func (b *nextKey) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&b.AccountID1, &b.AccountID2, &b.KeyOwner}
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	return nil
}

func (b nextKey) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{&b.AccountID1, &b.AccountID2, &b.KeyOwner}
	buf, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (b *membersFields) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	return nil
}

func (b membersFields) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	buf, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
