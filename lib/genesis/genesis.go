// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
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
	System   *System   `json:"System"`
	Babe     *Babe     `json:"babe"`
	Grandpa  *Grandpa  `json:"grandpa"`
	Balances *Balances `json:"Balances"`
	//TransactionPayment  interface{}         `json:"transactionPayment"`
	Sudo                *Sudo                `json:"sudo"`
	Session             *Session             `json:"session"`
	Staking             *Staking             `json:"staking"`
	Instance1Collective *Instance1Collective `json:"Instance1Collective"`
	Instance2Collective *Instance2Collective `json:"Instance2Collective"`
	PhragmenElection    *PhragmenElection    `json:"PhragmenElection"`
	Instance1Membership *Instance1Membership `json:"Instance1Membership"`
	Contracts           *Contracts           `json:"Contracts"`
	Society             *Society             `json:"Society"`
	Indices             *Indices             `json:"indices"`
	ImOnline            *ImOnline            `json:"imOnline"`
	AuthorityDiscovery  *AuthorityDiscovery  `json:"authorityDiscovery"`
	Vesting             *Vesting             `json:"vesting"`
	NominationPools     *NominationPools     `json:"nominationPools"`
	Configuration       *Configuration       `json:"configuration"`
	Paras               *Paras               `json:"paras"`
	Hrmp                *Hrmp                `json:"hrmp"`
	Registrar           *Registrar           `json:"registrar"`
	XcmPallet           *XcmPallet           `json:"xcmPallet"`
}

// System is ...
type System struct {
	Code string `json:"code"`
}

// Babe is ...
type Babe struct {
	Authorities []types.AuthorityAsAddress `json:"authorities"`
	EpochConfig *EpochConfig               `json:"epochConfig"`
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
	Balance   float64
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
	AccountID1 string
	AccountID2 string
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
	HistoryDepth          uint32         `json:"HistoryDepth"`
	ValidatorCount        uint32         `json:"validatorCount"`
	MinimumValidatorCount uint32         `json:"minimumValidatorCount"`
	Invulnerables         []string       `json:"invulnerables"`
	ForceEra              string         `json:"forceEra"`
	SlashRewardFraction   uint32         `json:"slashRewardFraction"`
	CanceledSlashPayout   *scale.Uint128 `json:"CanceledSlashPayout"`
	// CanceledPayout        int            `json:"canceledPayout"`
	// TODO: figure out below fields storage key. (#1868)
	// Stakers               [][]interface{} `json:"stakers"`
}

// Instance1Collective is ...
type Instance1Collective struct {
	Phantom interface{} `json:"Phantom"`
	Members []string    `json:"Members"`
}

// Instance2Collective is ...
type Instance2Collective struct {
	Phantom interface{}   `json:"Phantom"`
	Members []interface{} `json:"Members"`
}

// PhragmenElection is ...
type PhragmenElection struct {
	// TODO: figure out the correct encoding format of members data (#1866)
	Members []MembersFields `json:"Members"`
}

// MembersFields is ...
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
	Version            uint32             `json:"version"`
	EnablePrintln      bool               `json:"enable_println"`
	Limits             Limits             `json:"limits"`
	InstructionWeights InstructionWeights `json:"instruction_weights"`
	HostFnWeights      HostFnWeights      `json:"host_fn_weights"`
}

// Limits is ...
type Limits struct {
	EventTopics uint32 `json:"event_topics"`
	StackHeight uint32 `json:"stack_height"`
	Globals     uint32 `json:"globals"`
	Parameters  uint32 `json:"parameters"`
	MemoryPages uint32 `json:"memory_pages"`
	TableSize   uint32 `json:"table_size"`
	BrTableSize uint32 `json:"br_table_size"`
	SubjectLen  uint32 `json:"subject_len"`
	CodeSize    uint32 `json:"code_size"`
}

// InstructionWeights is ...
type InstructionWeights struct {
	I64Const             uint32 `json:"i64const"`
	I64Load              uint32 `json:"i64load"`
	I64Store             uint32 `json:"i64store"`
	Select               uint32 `json:"select"`
	If                   uint32 `json:"if"`
	Br                   uint32 `json:"br"`
	BrIf                 uint32 `json:"br_if"`
	BrTable              uint32 `json:"br_table"`
	BrTablePerEntry      uint32 `json:"br_table_per_entry"`
	Call                 uint32 `json:"call"`
	CallIndirect         uint32 `json:"call_indirect"`
	CallIndirectPerParam uint32 `json:"call_indirect_per_param"`
	LocalGet             uint32 `json:"local_get"`
	LocalSet             uint32 `json:"local_set"`
	LocalTee             uint32 `json:"local_tee"`
	GlobalGet            uint32 `json:"global_get"`
	GlobalSet            uint32 `json:"global_set"`
	MemoryCurrent        uint32 `json:"memory_current"`
	MemoryGrow           uint32 `json:"memory_grow"`
	I64Clz               uint32 `json:"i64clz"`
	I64Ctz               uint32 `json:"i64ctz"`
	I64Popcnt            uint32 `json:"i64popcnt"`
	I64Eqz               uint32 `json:"i64eqz"`
	I64Extendsi32        uint32 `json:"i64extendsi32"`
	I64Extendui32        uint32 `json:"i64extendui32"`
	I32Wrapi64           uint32 `json:"i32wrapi64"`
	I64Eq                uint32 `json:"i64eq"`
	I64Ne                uint32 `json:"i64ne"`
	I64Lts               uint32 `json:"i64lts"`
	I64Ltu               uint32 `json:"i64ltu"`
	I64Gts               uint32 `json:"i64gts"`
	I64Gtu               uint32 `json:"i64gtu"`
	I64Les               uint32 `json:"i64les"`
	I64Leu               uint32 `json:"i64leu"`
	I64Ges               uint32 `json:"i64ges"`
	I64Geu               uint32 `json:"i64geu"`
	I64Add               uint32 `json:"i64add"`
	I64Sub               uint32 `json:"i64sub"`
	I64Mul               uint32 `json:"i64mul"`
	I64Divs              uint32 `json:"i64divs"`
	I64Divu              uint32 `json:"i64divu"`
	I64Rems              uint32 `json:"i64rems"`
	I64Remu              uint32 `json:"i64remu"`
	I64And               uint32 `json:"i64and"`
	I64Or                uint32 `json:"i64or"`
	I64Xor               uint32 `json:"i64xor"`
	I64Shl               uint32 `json:"i64shl"`
	I64Shrs              uint32 `json:"i64shrs"`
	I64Shru              uint32 `json:"i64shru"`
	I64Rotl              uint32 `json:"i64rotl"`
	I64Rotr              uint32 `json:"i64rotr"`
}

// HostFnWeights is ...
type HostFnWeights struct {
	Caller                   uint64 `json:"caller"`
	Address                  uint64 `json:"address"`
	GasLeft                  uint64 `json:"gas_left"`
	Balance                  uint64 `json:"balance"`
	ValueTransferred         uint64 `json:"value_transferred"`
	MinimumBalance           uint64 `json:"minimum_balance"`
	TombstoneDeposit         uint64 `json:"tombstone_deposit"`
	RentAllowance            uint64 `json:"rent_allowance"`
	BlockNumber              uint64 `json:"block_number"`
	Now                      uint64 `json:"now"`
	WeightToFee              uint64 `json:"weight_to_fee"`
	Gas                      uint64 `json:"gas"`
	Input                    uint64 `json:"input"`
	InputPerByte             uint64 `json:"input_per_byte"`
	Return                   uint64 `json:"return"`
	ReturnPerByte            uint64 `json:"return_per_byte"`
	Terminate                uint64 `json:"terminate"`
	RestoreTo                uint64 `json:"restore_to"`
	RestoreToPerDelta        uint64 `json:"restore_to_per_delta"`
	Random                   uint64 `json:"random"`
	DepositEvent             uint64 `json:"deposit_event"`
	DepositEventPerTopic     uint64 `json:"deposit_event_per_topic"`
	DepositEventPerByte      uint64 `json:"deposit_event_per_byte"`
	SetRentAllowance         uint64 `json:"set_rent_allowance"`
	SetStorage               uint64 `json:"set_storage"`
	SetStoragePerByte        uint64 `json:"set_storage_per_byte"`
	ClearStorage             uint64 `json:"clear_storage"`
	GetStorage               uint64 `json:"get_storage"`
	GetStoragePerByte        uint64 `json:"get_storage_per_byte"`
	Transfer                 uint64 `json:"transfer"`
	Call                     uint64 `json:"call"`
	CallTransferSurcharge    uint64 `json:"call_transfer_surcharge"`
	CallPerInputByte         uint64 `json:"call_per_input_byte"`
	CallPerOutputByte        uint64 `json:"call_per_output_byte"`
	Instantiate              uint64 `json:"instantiate"`
	InstantiatePerInputByte  uint64 `json:"instantiate_per_input_byte"`
	InstantiatePerOutputByte uint64 `json:"instantiate_per_output_byte"`
	InstantiatePerSaltByte   uint64 `json:"instantiate_per_salt_byte"`
	HashSha2256              uint64 `json:"hash_sha2_256"`
	HashSha2256PerByte       uint64 `json:"hash_sha2_256_per_byte"`
	HashKeccak256            uint64 `json:"hash_keccak_256"`
	HashKeccak256PerByte     uint64 `json:"hash_keccak_256_per_byte"`
	HashBlake2256            uint64 `json:"hash_blake2_256"`
	HashBlake2256PerByte     uint64 `json:"hash_blake2_256_per_byte"`
	HashBlake2128            uint64 `json:"hash_blake2_128"`
	HashBlake2128PerByte     uint64 `json:"hash_blake2_128_per_byte"`
}

// Society is ...
type Society struct {
	Pot        *scale.Uint128 `json:"Pot"`
	MaxMembers uint32         `json:"MaxMembers"`
	Members    []string       `json:"Members"`
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

// UnmarshalJSON converts data to Go struct of type Runtime.
func (r *Runtime) UnmarshalJSON(data []byte) error {
	type runtimeTmp Runtime
	rtmt := runtimeTmp{}
	err := json.Unmarshal(data, &rtmt)
	*r = Runtime(rtmt)

	return err
}

// UnmarshalJSON converts data to Go struct of type BalancesFields.
func (b *BalancesFields) UnmarshalJSON(buf []byte) error {
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
func (b BalancesFields) MarshalJSON() ([]byte, error) {
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

// UnmarshalJSON converts data to Go struct of type MembersFields.
func (b *MembersFields) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return fmt.Errorf("error in MembersFields unmarshal: %w", err)
	}
	fmt.Println("MembersFields ===> ", b)
	if newLen := len(tmp); newLen != wantLen {
		return fmt.Errorf("wrong number of fields in MembersFields: %d != %d", newLen, wantLen)
	}
	return nil
}

// MarshalJSON converts Go struct of type MembersFields to []byte.
func (b MembersFields) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{&b.AccountID, &b.Balance}
	wantLen := len(tmp)

	buf, err := json.Marshal(tmp)
	if err != nil {
		return nil, fmt.Errorf("error in MembersFields marshal: %w", err)
	}
	if newLen := len(tmp); newLen != wantLen {
		return nil, fmt.Errorf("wrong number of fields in MembersFields: %d != %d", newLen, wantLen)
	}
	return buf, nil
}

// UnmarshalJSON converts data to Go struct of type NextKeys.
func (b *NextKeys) UnmarshalJSON(buf []byte) error {
	fmt.Println("buf ===> ", string(buf))

	tmp := []interface{}{&b.AccountID1, &b.AccountID2, &b.KeyOwner}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return fmt.Errorf("error in NextKeys unmarshal: %w", err)
	}
	fmt.Println("NextKeys ===> ", b)
	if newLen := len(tmp); newLen != wantLen {
		return fmt.Errorf("wrong number of fields in NextKeys: %d != %d", newLen, wantLen)
	}
	return nil
}

// MarshalJSON converts Go struct of type NextKeys to []byte.
func (b NextKeys) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{&b.AccountID1, &b.AccountID2, &b.KeyOwner}
	wantLen := len(tmp)

	buf, err := json.Marshal(tmp)
	if err != nil {
		return nil, fmt.Errorf("error in NextKeys marshal: %w", err)
	}
	if newLen := len(tmp); newLen != wantLen {
		return nil, fmt.Errorf("wrong number of fields in NextKeys: %d != %d", newLen, wantLen)
	}
	return buf, nil
}
