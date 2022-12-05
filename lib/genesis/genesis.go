// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"encoding/json"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Genesis stores the data parsed from the genesis configuration file
type Genesis struct {
	Name               string                 `json:"name"`
	ID                 string                 `json:"id"`
	ChainType          string                 `json:"chainType"`
	Bootnodes          []string               `json:"bootNodes"`
	TelemetryEndpoints []interface{}          `json:"telemetryEndpoints"`
	ProtocolID         string                 `json:"protocolId"`
	Genesis            Fields                 `json:"genesis"`
	Properties         map[string]interface{} `json:"properties"`
	ForkBlocks         []string               `json:"forkBlocks"`
	BadBlocks          []string               `json:"badBlocks"`
	ConsensusEngine    string                 `json:"consensusEngine"`
	CodeSubstitutes    map[string]string      `json:"codeSubstitutes"`
}

// Data defines the genesis file data formatted for trie storage
type Data struct {
	Name               string
	ID                 string
	ChainType          string
	Bootnodes          [][]byte
	TelemetryEndpoints []*TelemetryEndpoint
	ProtocolID         string
	Properties         map[string]interface{}
	ForkBlocks         []string
	BadBlocks          []string
	ConsensusEngine    string
	CodeSubstitutes    map[string]string
}

// TelemetryEndpoint struct to hold telemetry endpoint information
type TelemetryEndpoint struct {
	Endpoint  string
	Verbosity int
}

// Fields stores genesis raw data, and human readable runtime data
type Fields struct {
	Raw     map[string]map[string]string `json:"raw,omitempty"`
	Runtime *Runtime                     `json:"runtime,omitempty"`
}

// Runtime is ...
type Runtime struct {
	System              *System              `json:"System"`
	Babe                *babe                `json:"babe"`
	Grandpa             *grandpa             `json:"grandpa"`
	Balances            *balances            `json:"Balances"`
	Sudo                *sudo                `json:"sudo"`
	Session             *session             `json:"session"`
	Staking             *staking             `json:"staking"`
	Instance1Collective *instance1Collective `json:"Instance1Collective"`
	Instance2Collective *instance2Collective `json:"Instance2Collective"`
	PhragmenElection    *phragmenElection    `json:"PhragmenElection"`
	Instance1Membership *instance1Membership `json:"Instance1Membership"`
	Contracts           *contracts           `json:"Contracts"`
	Society             *society             `json:"Society"`
	Indices             *indices             `json:"indices"`
	ImOnline            *imOnline            `json:"imOnline"`
	AuthorityDiscovery  *authorityDiscovery  `json:"authorityDiscovery"`
	Vesting             *vesting             `json:"vesting"`
	NominationPools     *nominationPools     `json:"nominationPools"`
	Configuration       *configuration       `json:"configuration"`
	Paras               *paras               `json:"paras"`
	Hrmp                *hrmp                `json:"hrmp"`
	Registrar           *registrar           `json:"registrar"`
	XcmPallet           *xcmPallet           `json:"xcmPallet"`
}

// System is ...
type System struct {
	Code string `json:"code"`
}

type babe struct {
	Authorities []types.AuthorityAsAddress `json:"authorities"`
	EpochConfig *epochConfig               `json:"epochConfig"`
}

type epochConfig struct {
	C            []int  `json:"c"`
	AllowedSlots string `json:"allowed_slots"`
}

type grandpa struct {
	Authorities []types.AuthorityAsAddress `json:"authorities"`
}

type balances struct {
	Balances []balancesFields `json:"balances"`
}

type balancesFields struct {
	AccountID string
	Balance   float64
}

type sudo struct {
	Key string `json:"key"`
}

type indices struct {
	Indices []interface{} `json:"indices"`
}

type imOnline struct {
	Keys []interface{} `json:"keys"`
}

type authorityDiscovery struct {
	Keys []interface{} `json:"keys"`
}

type vesting struct {
	Vesting []interface{} `json:"vesting"`
}

type nominationPools struct {
	MinJoinBond       int64 `json:"minJoinBond"`
	MinCreateBond     int64 `json:"minCreateBond"`
	MaxPools          int   `json:"maxPools"`
	MaxMembersPerPool int   `json:"maxMembersPerPool"`
	MaxMembers        int   `json:"maxMembers"`
}

type configuration struct {
	Config config `json:"config"`
}

type config struct {
	MaxCodeSize                           int         `json:"max_code_size"`
	MaxHeadDataSize                       int         `json:"max_head_data_size"`
	MaxUpwardQueueCount                   int         `json:"max_upward_queue_count"`
	MaxUpwardQueueSize                    int         `json:"max_upward_queue_size"`
	MaxUpwardMessageSize                  int         `json:"max_upward_message_size"`
	MaxUpwardMessageNumPerCandidate       int         `json:"max_upward_message_num_per_candidate"`
	HrmpMaxMessageNumPerCandidate         int         `json:"hrmp_max_message_num_per_candidate"`
	ValidationUpgradeCooldown             int         `json:"validation_upgrade_cooldown"`
	ValidationUpgradeDelay                int         `json:"validation_upgrade_delay"`
	MaxPovSize                            int         `json:"max_pov_size"`
	MaxDownwardMessageSize                int         `json:"max_downward_message_size"`
	UmpServiceTotalWeight                 int64       `json:"ump_service_total_weight"`
	HrmpMaxParachainOutboundChannels      int         `json:"hrmp_max_parachain_outbound_channels"`
	HrmpMaxParathreadOutboundChannels     int         `json:"hrmp_max_parathread_outbound_channels"`
	HrmpSenderDeposit                     int         `json:"hrmp_sender_deposit"`
	HrmpRecipientDeposit                  int         `json:"hrmp_recipient_deposit"`
	HrmpChannelMaxCapacity                int         `json:"hrmp_channel_max_capacity"`
	HrmpChannelMaxTotalSize               int         `json:"hrmp_channel_max_total_size"`
	HrmpMaxParachainInboundChannels       int         `json:"hrmp_max_parachain_inbound_channels"`
	HrmpMaxParathreadInboundChannels      int         `json:"hrmp_max_parathread_inbound_channels"`
	HrmpChannelMaxMessageSize             int         `json:"hrmp_channel_max_message_size"`
	CodeRetentionPeriod                   int         `json:"code_retention_period"`
	ParathreadCores                       int         `json:"parathread_cores"`
	ParathreadRetries                     int         `json:"parathread_retries"`
	GroupRotationFrequency                int         `json:"group_rotation_frequency"`
	ChainAvailabilityPeriod               int         `json:"chain_availability_period"`
	ThreadAvailabilityPeriod              int         `json:"thread_availability_period"`
	SchedulingLookahead                   int         `json:"scheduling_lookahead"`
	MaxValidatorsPerCore                  interface{} `json:"max_validators_per_core"`
	MaxValidators                         interface{} `json:"max_validators"`
	DisputePeriod                         int         `json:"dispute_period"`
	DisputePostConclusionAcceptancePeriod int         `json:"dispute_post_conclusion_acceptance_period"`
	DisputeMaxSpamSlots                   int         `json:"dispute_max_spam_slots"`
	DisputeConclusionByTimeOutPeriod      int         `json:"dispute_conclusion_by_time_out_period"`
	NoShowSlots                           int         `json:"no_show_slots"`
	NDelayTranches                        int         `json:"n_delay_tranches"`
	ZerothDelayTrancheWidth               int         `json:"zeroth_delay_tranche_width"`
	NeededApprovals                       int         `json:"needed_approvals"`
	RelayVrfModuloSamples                 int         `json:"relay_vrf_modulo_samples"`
	UmpMaxIndividualWeight                int64       `json:"ump_max_individual_weight"`
	PvfCheckingEnabled                    bool        `json:"pvf_checking_enabled"`
	PvfVotingTTL                          int         `json:"pvf_voting_ttl"`
	MinimumValidationUpgradeDelay         int         `json:"minimum_validation_upgrade_delay"`
}

type paras struct {
	Paras []interface{} `json:"paras"`
}

type hrmp struct {
	PreopenHrmpChannels []interface{} `json:"preopenHrmpChannels"`
}

type registrar struct {
	NextFreeParaID int `json:"nextFreeParaId"`
}

type xcmPallet struct {
	SafeXcmVersion int `json:"safeXcmVersion"`
}

// GenesisData formats genesis for trie storage
func (g *Genesis) GenesisData() *Data {
	return &Data{
		Name:               g.Name,
		ID:                 g.ID,
		ChainType:          g.ChainType,
		Bootnodes:          common.StringArrayToBytes(g.Bootnodes),
		TelemetryEndpoints: interfaceToTelemetryEndpoint(g.TelemetryEndpoints),
		ProtocolID:         g.ProtocolID,
		Properties:         g.Properties,
		ForkBlocks:         g.ForkBlocks,
		BadBlocks:          g.BadBlocks,
		ConsensusEngine:    g.ConsensusEngine,
		CodeSubstitutes:    g.CodeSubstitutes,
	}
}

// GenesisFields returns the genesis fields including genesis raw data
func (g *Genesis) GenesisFields() Fields {
	return g.Genesis
}

// IsRaw returns whether the genesis is raw or not
func (g *Genesis) IsRaw() bool {
	return g.Genesis.Raw != nil || g.Genesis.Runtime == nil
}

// ToRaw converts a non-raw genesis to a raw genesis
func (g *Genesis) ToRaw() error {
	if g.IsRaw() {
		return nil
	}

	grt := g.Genesis.Runtime
	res, err := buildRawMap(*grt)
	if err != nil {
		return err
	}

	g.Genesis.Raw = make(map[string]map[string]string)
	g.Genesis.Raw["top"] = res
	return nil
}

func interfaceToTelemetryEndpoint(endpoints []interface{}) []*TelemetryEndpoint {
	var res []*TelemetryEndpoint
	for _, v := range endpoints {
		epi, ok := v.([]interface{})
		if !ok {
			continue
		}
		if len(epi) != 2 {
			continue
		}
		eps, ok := epi[0].(string)
		if !ok {
			continue
		}
		epv, ok := epi[1].(float64)
		if !ok {
			continue
		}
		ep := &TelemetryEndpoint{
			Endpoint:  eps,
			Verbosity: int(epv),
		}
		res = append(res, ep)
	}

	return res
}

// UnmarshalJSON converts data to Go struct of type BalancesFields.
func (b *balancesFields) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return fmt.Errorf("error in BalancesFields unmarshal: %w", err)
	}
	if newLen := len(tmp); newLen != wantLen {
		return fmt.Errorf("wrong number of fields in BalancesFields: %d != %d", newLen, wantLen)
	}
	return nil
}

// MarshalJSON converts Go struct of type BalancesFields to []byte.
func (b balancesFields) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	wantLen := len(tmp)

	buf, err := json.Marshal(tmp)
	if err != nil {
		return nil, fmt.Errorf("error in BalancesFields marshal: %w", err)
	}
	if newLen := len(tmp); newLen != wantLen {
		return nil, fmt.Errorf("wrong number of fields in BalancesFields: %d != %d", newLen, wantLen)
	}
	return buf, nil
}
