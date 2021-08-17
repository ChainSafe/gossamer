package genesis

import "github.com/ChainSafe/gossamer/pkg/scale"

type contracts struct {
	CurrentSchedule struct {
		Version       uint32 `json:"version"`
		EnablePrintln bool   `json:"enable_println"`
		Limits        struct {
			EventTopics uint32 `json:"event_topics"`
			StackHeight uint32 `json:"stack_height"`
			Globals     uint32 `json:"globals"`
			Parameters  uint32 `json:"parameters"`
			MemoryPages uint32 `json:"memory_pages"`
			TableSize   uint32 `json:"table_size"`
			BrTableSize uint32 `json:"br_table_size"`
			SubjectLen  uint32 `json:"subject_len"`
			CodeSize    uint32 `json:"code_size"`
		} `json:"limits"`
		InstructionWeights struct {
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
		} `json:"instruction_weights"`
		HostFnWeights struct {
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
		} `json:"host_fn_weights"`
	} `json:"CurrentSchedule"`
}

type society struct {
	Pot        *scale.Uint128 `json:"Pot"`
	MaxMembers uint32         `json:"MaxMembers"`
	// TODO: figure out the correct encoding format of members field
	Members []string `json:"Members"`
}

type staking struct {
	HistoryDepth          uint32         `json:"HistoryDepth"`
	ValidatorCount        uint32         `json:"ValidatorCount"`
	MinimumValidatorCount uint32         `json:"MinimumValidatorCount"`
	Invulnerables         []string       `json:"Invulnerables"`
	ForceEra              string         `json:"ForceEra"`
	SlashRewardFraction   uint32         `json:"SlashRewardFraction"`
	CanceledSlashPayout   *scale.Uint128 `json:"CanceledSlashPayout"`
	// TODO: figure out below fields storage key.
	// Stakers               [][]interface{} `json:"Stakers"`
}

type session struct {
	NextKeys [][]interface{} `json:"NextKeys"`
}

type instance1Collective struct {
	Phantom interface{}   `json:"Phantom"`
	Members []interface{} `json:"Members"`
}

type instance2Collective struct {
	Phantom interface{}   `json:"Phantom"`
	Members []interface{} `json:"Members"`
}

type instance1Membership struct {
	Phantom interface{}   `json:"Phantom"`
	Members []interface{} `json:"Members"`
}

type phragmenElection struct {
	// TODO: figure out the correct encoding format of members data
	Members [][]interface{} `json:"Members"`
}
