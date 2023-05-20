// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"encoding/json"

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

// Runtime is the structure of the genesis runtime field.
type Runtime struct {
	System             *System             `json:"system,omitempty"`
	Babe               *babe               `json:"babe,omitempty"`
	Grandpa            *grandpa            `json:"grandpa,omitempty"` //
	Balances           *balances           `json:"balances,omitempty"`
	Sudo               *sudo               `json:"sudo,omitempty"`    //
	Session            *session            `json:"session,omitempty"` //
	Staking            *staking            `json:"staking,omitempty"` //
	Indices            *indices            `json:"indices,omitempty"`
	ImOnline           *imOnline           `json:"imOnline,omitempty"`           //
	AuthorityDiscovery *authorityDiscovery `json:"authorityDiscovery,omitempty"` //
	Vesting            *vesting            `json:"vesting,omitempty"`            //
	NominationPools    *nominationPools    `json:"nominationPools,omitempty"`    //
	Configuration      *configuration      `json:"configuration,omitempty"`      //
	Paras              *paras              `json:"paras"`                        //
	Hrmp               *hrmp               `json:"hrmp"`                         //
	Registrar          *registrar          `json:"registrar,omitempty"`          //
	XcmPallet          *xcmPallet          `json:"xcmPallet,omitempty"`          //
}

// System is the system structure inside the runtime field for the genesis.
type System struct {
	Code string `json:"code,omitempty"`
}

type babe struct {
	Authorities []types.AuthorityAsAddress `json:"authorities,omitempty"`
	EpochConfig *epochConfig               `json:"epochConfig,omitempty"`
}

type epochConfig struct {
	C            []int  `json:"c,omitempty"`
	AllowedSlots string `json:"allowed_slots,omitempty"`
}

type grandpa struct {
	Authorities []types.AuthorityAsAddress `json:"authorities,omitempty"`
}

type balances struct {
	Balances []balancesFields `json:"balances,omitempty"`
}

type balancesFields struct {
	AccountID string
	Balance   float64
}

type sudo struct {
	Key string `json:"key,omitempty"`
}

type indices struct {
	Indices []interface{} `json:"indices,omitempty"`
}

type imOnline struct {
	Keys []interface{} `json:"keys,omitempty"`
}

type authorityDiscovery struct {
	Keys []interface{} `json:"keys,omitempty"`
}

type vesting struct {
	Vesting []interface{} `json:"vesting,omitempty"`
}

type nominationPools struct {
	MinJoinBond       int64 `json:"minJoinBond,omitempty"`
	MinCreateBond     int64 `json:"minCreateBond,omitempty"`
	MaxPools          int   `json:"maxPools,omitempty"`
	MaxMembersPerPool int   `json:"maxMembersPerPool,omitempty"`
	MaxMembers        int   `json:"maxMembers,omitempty"`
}

type configuration struct {
	Config config `json:"config,omitempty"`
}

type config struct {
	ChainAvailabilityPeriod               uint `json:"chain_availability_period,omitempty"`
	CodeRetentionPeriod                   uint `json:"code_retention_period,omitempty"`
	DisputeConclusionByTimeOutPeriod      uint `json:"dispute_conclusion_by_time_out_period,omitempty"`
	DisputeMaxSpamSlots                   uint `json:"dispute_max_spam_slots,omitempty"`
	DisputePeriod                         uint `json:"dispute_period,omitempty"`
	DisputePostConclusionAcceptancePeriod uint `json:"dispute_post_conclusion_acceptance_period,omitempty"`
	GroupRotationFrequency                uint `json:"group_rotation_frequency,omitempty"`
	HrmpChannelMaxCapacity                uint `json:"hrmp_channel_max_capacity,omitempty"`
	HrmpChannelMaxMessageSize             uint `json:"hrmp_channel_max_message_size,omitempty"`
	HrmpChannelMaxTotalSize               uint `json:"hrmp_channel_max_total_size,omitempty"`
	HrmpMaxMessageNumPerCandidate         uint `json:"hrmp_max_message_num_per_candidate,omitempty"`
	HrmpMaxParachainInboundChannels       uint `json:"hrmp_max_parachain_inbound_channels,omitempty"`
	HrmpMaxParachainOutboundChannels      uint `json:"hrmp_max_parachain_outbound_channels,omitempty"`
	HrmpMaxParathreadInboundChannels      uint `json:"hrmp_max_parathread_inbound_channels,omitempty"`
	HrmpMaxParathreadOutboundChannels     uint `json:"hrmp_max_parathread_outbound_channels,omitempty"`
	HrmpRecipientDeposit                  uint `json:"hrmp_recipient_deposit,omitempty"`
	HrmpSenderDeposit                     uint `json:"hrmp_sender_deposit,omitempty"`
	MaxCodeSize                           uint `json:"max_code_size,omitempty"`
	MaxDownwardMessageSize                uint `json:"max_downward_message_size,omitempty"`
	MaxHeadDataSize                       uint `json:"max_head_data_size,omitempty"`
	MaxPovSize                            uint `json:"max_pov_size,omitempty"`
	MaxUpwardMessageNumPerCandidate       uint `json:"max_upward_message_num_per_candidate,omitempty"`
	MaxUpwardMessageSize                  uint `json:"max_upward_message_size,omitempty"`
	MaxUpwardQueueCount                   uint `json:"max_upward_queue_count,omitempty"`
	MaxUpwardQueueSize                    uint `json:"max_upward_queue_size,omitempty"`
	MaxValidators                         uint `json:"max_validators,omitempty"`
	MaxValidatorsPerCore                  uint `json:"max_validators_per_core,omitempty"`
	MinimumValidationUpgradeDelay         uint `json:"minimum_validation_upgrade_delay,omitempty"`
	NDelayTranches                        uint `json:"n_delay_tranches,omitempty"`
	NeededApprovals                       uint `json:"needed_approvals,omitempty"`
	NoShowSlots                           uint `json:"no_show_slots,omitempty"`
	ParathreadCores                       uint `json:"parathread_cores,omitempty"`
	ParathreadRetries                     uint `json:"parathread_retries,omitempty"`
	PvfCheckingEnabled                    bool `json:"pvf_checking_enabled,omitempty"`
	PvfVotingTTL                          uint `json:"pvf_voting_ttl,omitempty"`
	RelayVrfModuloSamples                 uint `json:"relay_vrf_modulo_samples,omitempty"`
	SchedulingLookahead                   uint `json:"scheduling_lookahead,omitempty"`
	ThreadAvailabilityPeriod              uint `json:"thread_availability_period,omitempty"`
	UmpMaxIndividualWeight                struct {
		ProofSize uint `json:"proof_size,omitempty"`
		RefTime   uint `json:"ref_time,omitempty"`
	} `json:"ump_max_individual_weight,omitempty"`
	UmpServiceTotalWeight struct {
		ProofSize uint `json:"proof_size,omitempty"`
		RefTime   uint `json:"ref_time,omitempty"`
	} `json:"ump_service_total_weight,omitempty"`
	ValidationUpgradeCooldown uint `json:"validation_upgrade_cooldown,omitempty"`
	ValidationUpgradeDelay    uint `json:"validation_upgrade_delay,omitempty"`
	ZerothDelayTrancheWidth   uint `json:"zeroth_delay_tranche_width,omitempty"`
}

type paras struct {
	Paras []interface{} `json:"paras,omitempty"`
}

type hrmp struct {
	PreopenHrmpChannels []interface{} `json:"preopenHrmpChannels,omitempty"`
}

type registrar struct {
	NextFreeParaID int `json:"nextFreeParaId,omitempty"`
}

type xcmPallet struct {
	SafeXcmVersion int `json:"safeXcmVersion,omitempty"`
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

func (b *balancesFields) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	return nil
}

func (b balancesFields) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	buf, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
