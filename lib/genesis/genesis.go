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
	ForkID             string                 `json:"forkId"`
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
	ForkID             string
	ProtocolID         string
	Properties         map[string]interface{}
	ForkBlocks         []string
	BadBlocks          []string
	ConsensusEngine    string
	CodeSubstitutes    map[string]string
}

// TelemetryEndpoint struct to hold telemetry endpoint information
type TelemetryEndpoint struct {
	Endpoint  string `mapstructure:",squash"`
	Verbosity int    `mapstructure:",squash"`
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
	Grandpa            *grandpa            `json:"grandpa,omitempty"`
	Balances           *balances           `json:"balances,omitempty"`
	Sudo               *sudo               `json:"sudo,omitempty"`
	Session            *session            `json:"session,omitempty"`
	Staking            *staking            `json:"staking,omitempty"`
	Indices            *indices            `json:"indices,omitempty"`
	ImOnline           *imOnline           `json:"imOnline,omitempty"`
	AuthorityDiscovery *authorityDiscovery `json:"authorityDiscovery,omitempty"`
	Vesting            *vesting            `json:"vesting"`
	NominationPools    *nominationPools    `json:"nominationPools,omitempty"`
	Configuration      *configuration      `json:"configuration,omitempty"`
	Paras              *paras              `json:"paras"`
	Hrmp               *hrmp               `json:"hrmp"`
	Registrar          *registrar          `json:"registrar,omitempty"`
	XcmPallet          *xcmPallet          `json:"xcmPallet,omitempty"`
}

// System is the system structure inside the runtime field for the genesis.
type System struct {
	Code string `json:"code,omitempty"`
}

type babe struct {
	Authorities []types.AuthorityAsAddress `json:"authorities"`
	EpochConfig *epochConfig               `json:"epochConfig,omitempty"`
}

type epochConfig struct {
	C            []int  `json:"c,omitempty"`
	AllowedSlots string `json:"allowed_slots,omitempty"`
}

type grandpa struct {
	Authorities []types.AuthorityAsAddress `json:"authorities"`
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
	Indices []interface{} `json:"indices"`
}

type imOnline struct {
	Keys []string `json:"keys"`
}

type authorityDiscovery struct {
	Keys []string `json:"keys"`
}

type vesting struct {
	Vesting []interface{} `json:"vesting"`
}

type nominationPools struct {
	MinJoinBond       *uint `json:"minJoinBond,omitempty"`
	MinCreateBond     *uint `json:"minCreateBond,omitempty"`
	MaxPools          *uint `json:"maxPools,omitempty"`
	MaxMembersPerPool *uint `json:"maxMembersPerPool,omitempty"`
	MaxMembers        *uint `json:"maxMembers,omitempty"`
}

type configuration struct {
	Config config `json:"config,omitempty"`
}

type config struct {
	MaxCodeSize                     *uint `json:"max_code_size"`
	MaxHeadDataSize                 *uint `json:"max_head_data_size"`
	MaxUpwardQueueCount             *uint `json:"max_upward_queue_count"`
	MaxUpwardQueueSize              *uint `json:"max_upward_queue_size"`
	MaxUpwardMessageSize            *uint `json:"max_upward_message_size"`
	MaxUpwardMessageNumPerCandidate *uint `json:"max_upward_message_num_per_candidate"`
	HrmpMaxMessageNumPerCandidate   *uint `json:"hrmp_max_message_num_per_candidate"`
	ValidationUpgradeCooldown       *uint `json:"validation_upgrade_cooldown"`
	ValidationUpgradeDelay          *uint `json:"validation_upgrade_delay"`
	MaxPovSize                      *uint `json:"max_pov_size"`
	MaxDownwardMessageSize          *uint `json:"max_downward_message_size"`
	UmpServiceTotalWeight           *struct {
		RefTime   *uint `json:"ref_time"`
		ProofSize *uint `json:"proof_size"`
	} `json:"ump_service_total_weight"`
	HrmpMaxParachainOutboundChannels      *uint `json:"hrmp_max_parachain_outbound_channels"`
	HrmpMaxParathreadOutboundChannels     *uint `json:"hrmp_max_parathread_outbound_channels"`
	HrmpSenderDeposit                     *uint `json:"hrmp_sender_deposit"`
	HrmpRecipientDeposit                  *uint `json:"hrmp_recipient_deposit"`
	HrmpChannelMaxCapacity                *uint `json:"hrmp_channel_max_capacity"`
	HrmpChannelMaxTotalSize               *uint `json:"hrmp_channel_max_total_size"`
	HrmpMaxParachainInboundChannels       *uint `json:"hrmp_max_parachain_inbound_channels"`
	HrmpMaxParathreadInboundChannels      *uint `json:"hrmp_max_parathread_inbound_channels"`
	HrmpChannelMaxMessageSize             *uint `json:"hrmp_channel_max_message_size"`
	CodeRetentionPeriod                   *uint `json:"code_retention_period"`
	ParathreadCores                       *uint `json:"parathread_cores"`
	ParathreadRetries                     *uint `json:"parathread_retries"`
	GroupRotationFrequency                *uint `json:"group_rotation_frequency"`
	ChainAvailabilityPeriod               *uint `json:"chain_availability_period"`
	ThreadAvailabilityPeriod              *uint `json:"thread_availability_period"`
	SchedulingLookahead                   *uint `json:"scheduling_lookahead"`
	MaxValidatorsPerCore                  *uint `json:"max_validators_per_core"`
	MaxValidators                         *uint `json:"max_validators"`
	DisputePeriod                         *uint `json:"dispute_period"`
	DisputePostConclusionAcceptancePeriod *uint `json:"dispute_post_conclusion_acceptance_period"`
	DisputeMaxSpamSlots                   *uint `json:"dispute_max_spam_slots"`
	DisputeConclusionByTimeOutPeriod      *uint `json:"dispute_conclusion_by_time_out_period"`
	NoShowSlots                           *uint `json:"no_show_slots"`
	NDelayTranches                        *uint `json:"n_delay_tranches"`
	ZerothDelayTrancheWidth               *uint `json:"zeroth_delay_tranche_width"`
	NeededApprovals                       *uint `json:"needed_approvals"`
	RelayVrfModuloSamples                 *uint `json:"relay_vrf_modulo_samples"`
	UmpMaxIndividualWeight                *struct {
		RefTime   *uint `json:"ref_time"`
		ProofSize *uint `json:"proof_size"`
	} `json:"ump_max_individual_weight"`
	PvfCheckingEnabled            bool  `json:"pvf_checking_enabled"`
	PvfVotingTTL                  *uint `json:"pvf_voting_ttl"`
	MinimumValidationUpgradeDelay *uint `json:"minimum_validation_upgrade_delay"`
}

type paras struct {
	Paras []interface{} `json:"paras"`
}

type hrmp struct {
	PreopenHrmpChannels []interface{} `json:"preopenHrmpChannels"`
}

type registrar struct {
	NextFreeParaID *uint `json:"nextFreeParaId,omitempty"`
}

type xcmPallet struct {
	SafeXcmVersion *uint `json:"safeXcmVersion,omitempty"`
}

// GenesisData formats genesis for trie storage
func (g *Genesis) GenesisData() *Data {
	return &Data{
		Name:               g.Name,
		ID:                 g.ID,
		ChainType:          g.ChainType,
		Bootnodes:          common.StringArrayToBytes(g.Bootnodes),
		TelemetryEndpoints: interfaceToTelemetryEndpoint(g.TelemetryEndpoints),
		ForkID:             g.ForkID,
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
	return json.Unmarshal(buf, &tmp)
}

func (b balancesFields) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	buf, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
