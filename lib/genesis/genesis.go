// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"encoding/json"
	"fmt"
	"math/big"

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
	System   *System  `json:"system"`
	Babe     Babe     `json:"babe"`
	Grandpa  Grandpa  `json:"grandpa"`
	Balances Balances `json:"balances"`
	//TransactionPayment  interface{}         `json:"transactionPayment"`
	Sudo                Sudo                `json:"sudo"`
	Session             Session             `json:"session"`
	Staking             Staking             `json:"staking"`
	Instance1Collective Instance1Collective `json:"Instance1Collective"`
	Instance2Collective Instance2Collective `json:"Instance2Collective"`
	PhragmenElection    PhragmenElection    `json:"PhragmenElection"`
	Instance1Membership Instance1Membership `json:"Instance1Membership"`
	Contracts           Contracts           `json:"Contracts"`
	Society             Society             `json:"Society"`
	Indices             Indices             `json:"indices"`
	ImOnline            ImOnline            `json:"imOnline"`
	AuthorityDiscovery  AuthorityDiscovery  `json:"authorityDiscovery"`
	Vesting             Vesting             `json:"vesting"`
	NominationPools     NominationPools     `json:"nominationPools"`
	Configuration       Configuration       `json:"configuration"`
	Paras               Paras               `json:"paras"`
	Hrmp                Hrmp                `json:"hrmp"`
	Registrar           Registrar           `json:"registrar"`
	XcmPallet           XcmPallet           `json:"xcmPallet"`
}

// System is ...
type System struct {
	Code string `json:"code"`
}

// Babe is ...
type Babe struct {
	Authorities []types.AuthorityAsAddress `json:"authorities"`
	EpochConfig EpochConfig                `json:"epochConfig"`
}

// EpochConfig is ...
type EpochConfig struct {
	C            []int  `json:"c"`
	AllowedSlots string `json:"allowed_slots"`
}

// Grandpa is ...
type Grandpa struct {
	Authorities []types.AuthorityAsAddress `json:"authorities"`
}

// Balances is ...
type Balances struct {
	Balances []BalancesFields `json:"balances"`
}

// BalancesFields is ...
type BalancesFields struct {
	AccountID string
	Balance   big.Int
}

// Sudo is ...
type Sudo struct {
	Key string `json:"key"`
}

// Session is ...
type Session struct {
	// Keys     [][]interface{} `json:"keys"`
	NextKeys []NextKeys `json:"NextKeys"`
}

// NextKeys is ...
type NextKeys struct {
	AccountId1 string
	AccountId2 string
	KeyOwner   KeyOwner
}

// KeyOwner is ...
type KeyOwner struct {
	Grandpa            string `json:"grandpa"`
	Babe               string `json:"babe"`
	ImOnline           string `json:"im_online"`
	AuthorityDiscovery string `json:"authority_discovery"`
}

// Staking is ...
type Staking struct {
	HistoryDepth          int      `json:"historyDepth"`
	ValidatorCount        int      `json:"validatorCount"`
	MinimumValidatorCount int      `json:"minimumValidatorCount"`
	Invulnerables         []string `json:"invulnerables"`
	ForceEra              string   `json:"forceEra"`
	SlashRewardFraction   int      `json:"slashRewardFraction"`
	CanceledPayout        int      `json:"canceledPayout"`
	CanceledSlashPayout   int      `json:"CanceledSlashPayout"`
	// TODO: figure out below fields storage key. (#1868)
	// Stakers               [][]interface{} `json:"stakers"`
	MinNominatorBond  int `json:"minNominatorBond"`
	MinValidatorBond  int `json:"minValidatorBond"`
	MaxValidatorCount int `json:"maxValidatorCount"`
	MaxNominatorCount int `json:"maxNominatorCount"`
}

// Instance1Collective is ...
type Instance1Collective struct {
	Phantom interface{} `json:"Phantom"`
	Members []string    `json:"Members"`
}

// Instance2Collective is ...
type Instance2Collective struct {
	Phantom interface{} `json:"Phantom"`
	Members []string    `json:"Members"`
}

// PhragmenElection is ...
type PhragmenElection struct {
	// TODO: figure out the correct encoding format of members data (#1866)
	Members []MembersFields `json:"Members"`
}

type MembersFields struct {
	AccountID string
	Balance   big.Int
}

// Instance1Membership is ...
type Instance1Membership struct {
	Members []string    `json:"Members"`
	Phantom interface{} `json:"Phantom"`
}

// Contracts is ...
type Contracts struct {
	CurrentSchedule CurrentSchedule `json:"CurrentSchedule"`
}

// CurrentSchedule is ...
type CurrentSchedule struct {
	Version            int                `json:"version"`
	EnablePrintln      bool               `json:"enable_println"`
	Limits             Limits             `json:"limits"`
	InstructionWeights InstructionWeights `json:"instruction_weights"`
	HostFnWeights      HostFnWeights      `json:"host_fn_weights"`
}

// Limits is ...
type Limits struct {
	EventTopics int `json:"event_topics"`
	StackHeight int `json:"stack_height"`
	Globals     int `json:"globals"`
	Parameters  int `json:"parameters"`
	MemoryPages int `json:"memory_pages"`
	TableSize   int `json:"table_size"`
	BrTableSize int `json:"br_table_size"`
	SubjectLen  int `json:"subject_len"`
	CodeSize    int `json:"code_size"`
}

// InstructionWeights is ...
type InstructionWeights struct {
	I64Const             int `json:"i64const"`
	I64Load              int `json:"i64load"`
	I64Store             int `json:"i64store"`
	Select               int `json:"select"`
	If                   int `json:"if"`
	Br                   int `json:"br"`
	BrIf                 int `json:"br_if"`
	BrTable              int `json:"br_table"`
	BrTablePerEntry      int `json:"br_table_per_entry"`
	Call                 int `json:"call"`
	CallIndirect         int `json:"call_indirect"`
	CallIndirectPerParam int `json:"call_indirect_per_param"`
	LocalGet             int `json:"local_get"`
	LocalSet             int `json:"local_set"`
	LocalTee             int `json:"local_tee"`
	GlobalGet            int `json:"global_get"`
	GlobalSet            int `json:"global_set"`
	MemoryCurrent        int `json:"memory_current"`
	MemoryGrow           int `json:"memory_grow"`
	I64Clz               int `json:"i64clz"`
	I64Ctz               int `json:"i64ctz"`
	I64Popcnt            int `json:"i64popcnt"`
	I64Eqz               int `json:"i64eqz"`
	I64Extendsi32        int `json:"i64extendsi32"`
	I64Extendui32        int `json:"i64extendui32"`
	I32Wrapi64           int `json:"i32wrapi64"`
	I64Eq                int `json:"i64eq"`
	I64Ne                int `json:"i64ne"`
	I64Lts               int `json:"i64lts"`
	I64Ltu               int `json:"i64ltu"`
	I64Gts               int `json:"i64gts"`
	I64Gtu               int `json:"i64gtu"`
	I64Les               int `json:"i64les"`
	I64Leu               int `json:"i64leu"`
	I64Ges               int `json:"i64ges"`
	I64Geu               int `json:"i64geu"`
	I64Add               int `json:"i64add"`
	I64Sub               int `json:"i64sub"`
	I64Mul               int `json:"i64mul"`
	I64Divs              int `json:"i64divs"`
	I64Divu              int `json:"i64divu"`
	I64Rems              int `json:"i64rems"`
	I64Remu              int `json:"i64remu"`
	I64And               int `json:"i64and"`
	I64Or                int `json:"i64or"`
	I64Xor               int `json:"i64xor"`
	I64Shl               int `json:"i64shl"`
	I64Shrs              int `json:"i64shrs"`
	I64Shru              int `json:"i64shru"`
	I64Rotl              int `json:"i64rotl"`
	I64Rotr              int `json:"i64rotr"`
}

// HostFnWeights is ...
type HostFnWeights struct {
	Caller                   int `json:"caller"`
	Address                  int `json:"address"`
	GasLeft                  int `json:"gas_left"`
	Balance                  int `json:"balance"`
	ValueTransferred         int `json:"value_transferred"`
	MinimumBalance           int `json:"minimum_balance"`
	TombstoneDeposit         int `json:"tombstone_deposit"`
	RentAllowance            int `json:"rent_allowance"`
	BlockNumber              int `json:"block_number"`
	Now                      int `json:"now"`
	WeightToFee              int `json:"weight_to_fee"`
	Gas                      int `json:"gas"`
	Input                    int `json:"input"`
	InputPerByte             int `json:"input_per_byte"`
	Return                   int `json:"return"`
	ReturnPerByte            int `json:"return_per_byte"`
	Terminate                int `json:"terminate"`
	RestoreTo                int `json:"restore_to"`
	RestoreToPerDelta        int `json:"restore_to_per_delta"`
	Random                   int `json:"random"`
	DepositEvent             int `json:"deposit_event"`
	DepositEventPerTopic     int `json:"deposit_event_per_topic"`
	DepositEventPerByte      int `json:"deposit_event_per_byte"`
	SetRentAllowance         int `json:"set_rent_allowance"`
	SetStorage               int `json:"set_storage"`
	SetStoragePerByte        int `json:"set_storage_per_byte"`
	ClearStorage             int `json:"clear_storage"`
	GetStorage               int `json:"get_storage"`
	GetStoragePerByte        int `json:"get_storage_per_byte"`
	Transfer                 int `json:"transfer"`
	Call                     int `json:"call"`
	CallTransferSurcharge    int `json:"call_transfer_surcharge"`
	CallPerInputByte         int `json:"call_per_input_byte"`
	CallPerOutputByte        int `json:"call_per_output_byte"`
	Instantiate              int `json:"instantiate"`
	InstantiatePerInputByte  int `json:"instantiate_per_input_byte"`
	InstantiatePerOutputByte int `json:"instantiate_per_output_byte"`
	InstantiatePerSaltByte   int `json:"instantiate_per_salt_byte"`
	HashSha2256              int `json:"hash_sha2_256"`
	HashSha2256PerByte       int `json:"hash_sha2_256_per_byte"`
	HashKeccak256            int `json:"hash_keccak_256"`
	HashKeccak256PerByte     int `json:"hash_keccak_256_per_byte"`
	HashBlake2256            int `json:"hash_blake2_256"`
	HashBlake2256PerByte     int `json:"hash_blake2_256_per_byte"`
	HashBlake2128            int `json:"hash_blake2_128"`
	HashBlake2128PerByte     int `json:"hash_blake2_128_per_byte"`
}

// Society is ...
type Society struct {
	Pot        int      `json:"Pot"`
	MaxMembers int      `json:"MaxMembers"`
	Members    []string `json:"Members"`
}

// Indices is ...
type Indices struct {
	Indices []interface{} `json:"indices"`
}

// ImOnline is ...
type ImOnline struct {
	Keys []interface{} `json:"keys"`
}

// AuthorityDiscovery is ...
type AuthorityDiscovery struct {
	Keys []interface{} `json:"keys"`
}

// Vesting is ...
type Vesting struct {
	Vesting []interface{} `json:"vesting"`
}

// NominationPools is ...
type NominationPools struct {
	MinJoinBond       int64 `json:"minJoinBond"`
	MinCreateBond     int64 `json:"minCreateBond"`
	MaxPools          int   `json:"maxPools"`
	MaxMembersPerPool int   `json:"maxMembersPerPool"`
	MaxMembers        int   `json:"maxMembers"`
}

// Configuration is ...
type Configuration struct {
	Config Config `json:"config"`
}

// Config is ...
type Config struct {
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

// Paras is ...
type Paras struct {
	Paras []interface{} `json:"paras"`
}

// Hrmp is ...
type Hrmp struct {
	PreopenHrmpChannels []interface{} `json:"preopenHrmpChannels"`
}

// Registrar is ...
type Registrar struct {
	NextFreeParaID int `json:"nextFreeParaId"`
}

//XcmPallet is ...
type XcmPallet struct {
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

// UnmarshalJSON method of the Runtime structure.
func (r *Runtime) UnmarshalJSON(data []byte) error {
	type runtimeTmp Runtime
	rtmt := runtimeTmp{}
	err := json.Unmarshal(data, &rtmt)
	*r = Runtime(rtmt)

	return err
}

//Custom Unmarshal method for BalancesFields.
func (b *BalancesFields) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return fmt.Errorf("error in balance marshal: %w", err)
	}
	if newLen := len(tmp); newLen != wantLen {
		return fmt.Errorf("wrong number of fields in BalancesFields: %d != %d", newLen, wantLen)
	}
	return nil
}

// Custom marshal method for AuthorityAsAddress.
func (b BalancesFields) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	wantLen := len(tmp)

	buf, err := json.Marshal(tmp)
	if err != nil {
		return nil, fmt.Errorf("error in balance marshal: %w", err)
	}
	if newLen := len(tmp); newLen != wantLen {
		return nil, fmt.Errorf("wrong number of fields in BalancesFields: %d != %d", newLen, wantLen)
	}
	return buf, nil
}
