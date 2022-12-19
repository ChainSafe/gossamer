// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestNewGenesisRawFromJSON(t *testing.T) {
	expected := &Genesis{
		Name: "gossamer",
		ID:   "gossamer",
		Bootnodes: []string{
			"/dns4/p2p.cc3-0.kusama.network/tcp/30100/p2p/QmeCit3Nif4VfNqrEJsdYHZGcKzRCnZvGxg6hha1iNj4mk",
			"/dns4/p2p.cc3-1.kusama.network/tcp/30100/p2p/QmchDJtEGiEWf7Ag58HNoTg9jSGzxkSZ23VgmF6xiLKKsZ",
		},
		TelemetryEndpoints: []interface{}{"wss://telemetry.polkadot.io/submit/", float64(1)},
		ProtocolID:         "/gossamer/test/0",
		Properties: map[string]interface{}{
			"ss58Format":    float64(0),
			"tokenDecimals": float64(10),
			"tokenSymbol":   "DOT",
		},
		ForkBlocks: []string{"fork1", "forkBlock2"},
		BadBlocks:  []string{"badBlock1", "badBlock2"},
		Genesis: Fields{
			Raw: map[string]map[string]string{
				"top": {"0x3a636f6465": "0x0102"},
			},
		},
	}

	// Grab json encoded bytes
	bz, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	// Write to temp file
	filename := filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(filename, bz, os.ModePerm)
	require.NoError(t, err)

	genesis, err := NewGenesisFromJSONRaw(filename)
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, expected, genesis)
}

//nolint:lll
func TestNewGenesisFromJSON(t *testing.T) {
	var expectedGenesis = &Genesis{}
	const zeroByte = "0x00"

	expRaw := make(map[string]map[string]string)
	expRaw["top"] = make(map[string]string)
	expRaw["top"]["0x3a636f6465"] = "0xfoo"                                                                                                                                                                                                                                                                                                            // raw system code
	expRaw["top"]["0x3a6772616e6470615f617574686f726974696573"] = "0x010834602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a6910000000000000000"                                                                                                                                                                                             // raw grandpa authorities
	expRaw["top"]["0x014f204c006a2837deb5551ba5211d6c5e0621c4869aa60c02be9adcc98a0d1d"] = "0x08d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000"                                                                                                                                                                       // raw babe authorities
	expRaw["top"]["0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x0000000000000000000000007aeb9049000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" // Balances
	expRaw["top"][common.BytesToHex(common.UpgradedToDualRefKey)] = "0x01"
	expRaw["top"]["0x426e15054d267946093858132eb537f1a47a9ff5cd5bf4d848a80a0b1a947dc3"] = "0x00000000000000000000000000000000"                                                                                                                 // Society
	expRaw["top"]["0x426e15054d267946093858132eb537f1ba7fb8745735dc3be2a2c61a72c39e78"] = "0x0101d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48"             // Society
	expRaw["top"]["0x426e15054d267946093858132eb537f1d0b4a3f7631f0c0e761898fe198211de"] = "0xe7030000"                                                                                                                                         // Society
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707138e71612491192d68deab7e6f563fe1"] = "0x02000000"                                                                                                                                         // Staking
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e7075579297f4dfb9609e7e4c2ebab9ce40a"] = "0x80be5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f"                                                                               // Staking
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707ac0a2cbf8e355f5ea6cb2de8727bfb0c"] = "0x54000000"                                                                                                                                         // Staking
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707b49a2738eeb30896aacb8b3fb46471bd"] = "0x01000000"                                                                                                                                         // Staking
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707c29a0310e1bb45d20cace77ccb62c97d"] = "0x00e1f505"                                                                                                                                         // Staking
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707f7dad0317324aecae8744b87fc95f2f3"] = "0x00e1f505"                                                                                                                                         // Staking
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707f7dad0317324aecae8744b87fc95f2f3"] = zeroByte                                                                                                                                             // Staking
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b54c014e6bf8b8c2c011e7290b85696bb3e535263148daaf49be5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f"] = "0xbe5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f" // Session
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa195066b8d48da86b869b6261626580d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa1950c9b0c13125732d276175646980d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa1950ed43a85541921049696d6f6e80d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa1950f5537bdb2a1f626b6772616e8088dc3417d5058ec4b4503e0c12ea1a0a89be200fe98922423d4334014fa6b0ee"] = "0x"                                                       // Session
	expRaw["top"]["0x11f3ba2e1cdd6d62f2ff9b5589e7ff81ba7fb8745735dc3be2a2c61a72c39e78"] = zeroByte                                                                                                                                             // Instance1Collective
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e70728dccb559b95c40168a1b2696581b5a7"] = "0x00000000000000000000000000000000"                                                                                                                 // Staking.CanceledSlashPayout
	expRaw["top"]["0x8985776095addd4789fccbce8ca77b23ba7fb8745735dc3be2a2c61a72c39e78"] = "0x08d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48"               // Instance2Collective
	expRaw["top"]["0x492a52699edf49c972c21db794cfcf57ba7fb8745735dc3be2a2c61a72c39e78"] = zeroByte                                                                                                                                             // Instance1Membership
	// Contract
	expRaw["top"]["0x4342193e496fab7ec59d615ed0dc5530d2d505c0e6f76fd7ce0796ebe187401c"] = "0x010000000001040000000002000000010000800000001000000000100000000100002000000000000800150600004c6b02004c8103009e1800000b1d0000160d0000651800006d2b00008a000000f5700100fdf602008e070000440600006e070000030600004f180000b528000091070000b60da70827080000590800006a0a0000ef070000560800004a0800008e080000f509000061090000dd090000a10a00009c090000e409000091090000650900001e0a0000120a0000ae09000099090000060a00006b2000000b1d000051200000221d000094090000ad090000b6090000160a0000660a0000fd090000260a0000440a0000d41a2a0000000000a0c729000000000092122900000000001ab5580000000000ba1c290000000000e000290000000000b0ef280000000000ee325c0000000000dec1280000000000ca07290000000000c07d4e00000000009c77140000000000303a7200000000000b01000000000000f0ab450000000000ff0200000000000060a21c270000000030078d31000000002635af09000000000ae164000000000038b18e0000000000b6b1cc0700000000890900000000000040036c00000000008ad6e21100000000de020000000000006e67cc080000000078f6110200000000e605000000000000acb1b50a00000000b24419090000000092579f08000000004702000000000000240300000000000016eaeb220000000055020000000000003503000000000000db0a000000000000f4802600000000006a100000000000006a9a280000000000220d0000000000004e9c2400000000001c0600000000000026832400000000001b06000000000000"
	expectedGenesis.Genesis = Fields{
		Raw: expRaw,
	}

	// create human readable test genesis
	testGenesis := &Genesis{}
	hrData := new(Runtime)
	hrData.System = &System{Code: "0xfoo"} // system code entry
	BabeAuth1 := types.AuthorityAsAddress{Address: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY", Weight: 1}
	hrData.Babe = &babe{Authorities: []types.AuthorityAsAddress{BabeAuth1}} // babe authority data
	GrandpaAuth1 := types.AuthorityAsAddress{Address: "5DFNv4Txc4b88qHqQ6GG4D646QcT4fN3jjS2G3r1PyZkfDut", Weight: 0}
	hrData.Grandpa = &grandpa{Authorities: []types.AuthorityAsAddress{GrandpaAuth1}} // grandpa authority data
	balConf1 := balancesFields{"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY", 1234234234}
	hrData.Balances = &balances{Balances: []balancesFields{balConf1}} // balances
	// Add test cases for new fields...

	zeroOfUint128 := scale.MustNewUint128(new(big.Int).SetUint64(0))

	hrData.Society = &society{
		Pot:        zeroOfUint128,
		MaxMembers: 999,
		Members: []string{
			"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
			"5FHneW46xGXgs5mUiveU4sbTyGBzmstUspZC92UhjJM694ty",
		}}

	hrData.Staking = &staking{
		HistoryDepth:          84,
		ValidatorCount:        2,
		MinimumValidatorCount: 1,
		ForceEra:              "NotForcing",
		SlashRewardFraction:   100000000,
		Invulnerables: []string{
			"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
		},
		CanceledSlashPayout: zeroOfUint128,
	}

	hrData.Session = &session{
		NextKeys: []nextKey{
			{
				"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
				"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
				keyOwner{
					Grandpa:            "5FA9nQDVg267DEd8m1ZypXLBnvN7SFxYwV7ndqSYGiN9TTpu",
					Babe:               "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
					ImOnline:           "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
					AuthorityDiscovery: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
				},
			},
		},
	}

	hrData.Instance1Collective = &instance1Collective{
		Phantom: nil,
		Members: []interface{}{},
	}
	hrData.Instance2Collective = &instance2Collective{
		Phantom: nil,
		Members: []interface{}{
			"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
			"5FHneW46xGXgs5mUiveU4sbTyGBzmstUspZC92UhjJM694ty",
		},
	}

	hrData.Instance1Membership = &instance1Membership{
		Phantom: nil,
		Members: []interface{}{},
	}

	hrData.Contracts = &contracts{
		CurrentSchedule: currentSchedule{
			Version:       0,
			EnablePrintln: true,
			Limits: limits{
				EventTopics: 4,
				StackHeight: 512,
				Globals:     256,
				Parameters:  128,
				MemoryPages: 16,
				TableSize:   4096,
				BrTableSize: 256,
				SubjectLen:  32,
				CodeSize:    524288,
			},
			InstructionWeights: instructionWeights{
				I64Const:             1557,
				I64Load:              158540,
				I64Store:             229708,
				Select:               6302,
				If:                   7435,
				Br:                   3350,
				BrIf:                 6245,
				BrTable:              11117,
				BrTablePerEntry:      138,
				Call:                 94453,
				CallIndirect:         194301,
				CallIndirectPerParam: 1934,
				LocalGet:             1604,
				LocalSet:             1902,
				LocalTee:             1539,
				GlobalGet:            6223,
				GlobalSet:            10421,
				MemoryCurrent:        1937,
				MemoryGrow:           145165750,
				I64Clz:               2087,
				I64Ctz:               2137,
				I64Popcnt:            2666,
				I64Eqz:               2031,
				I64Extendsi32:        2134,
				I64Extendui32:        2122,
				I32Wrapi64:           2190,
				I64Eq:                2549,
				I64Ne:                2401,
				I64Lts:               2525,
				I64Ltu:               2721,
				I64Gts:               2460,
				I64Gtu:               2532,
				I64Les:               2449,
				I64Leu:               2405,
				I64Ges:               2590,
				I64Geu:               2578,
				I64Add:               2478,
				I64Sub:               2457,
				I64Mul:               2566,
				I64Divs:              8299,
				I64Divu:              7435,
				I64Rems:              8273,
				I64Remu:              7458,
				I64And:               2452,
				I64Or:                2477,
				I64Xor:               2486,
				I64Shl:               2582,
				I64Shrs:              2662,
				I64Shru:              2557,
				I64Rotl:              2598,
				I64Rotr:              2628,
			},
			HostFnWeights: hostFnWeights{
				Caller:                   2759380,
				Address:                  2738080,
				GasLeft:                  2691730,
				Balance:                  5813530,
				ValueTransferred:         2694330,
				MinimumBalance:           2687200,
				TombstoneDeposit:         2682800,
				RentAllowance:            6042350,
				BlockNumber:              2671070,
				Now:                      2688970,
				WeightToFee:              5144000,
				Gas:                      1341340,
				Input:                    7486000,
				InputPerByte:             267,
				Return:                   4566000,
				ReturnPerByte:            767,
				Terminate:                656188000,
				RestoreTo:                831326000,
				RestoreToPerDelta:        162477350,
				Random:                   6611210,
				DepositEvent:             9351480,
				DepositEventPerTopic:     130855350,
				DepositEventPerByte:      2441,
				SetRentAllowance:         7078720,
				SetStorage:               300078730,
				SetStoragePerByte:        734,
				ClearStorage:             147613550,
				GetStorage:               34731640,
				GetStoragePerByte:        1510,
				Transfer:                 179679660,
				Call:                     152650930,
				CallTransferSurcharge:    144660370,
				CallPerInputByte:         583,
				CallPerOutputByte:        804,
				Instantiate:              585886230,
				InstantiatePerInputByte:  597,
				InstantiatePerOutputByte: 821,
				InstantiatePerSaltByte:   2779,
				HashSha2256:              2523380,
				HashSha2256PerByte:       4202,
				HashKeccak256:            2660970,
				HashKeccak256PerByte:     3362,
				HashBlake2256:            2399310,
				HashBlake2256PerByte:     1564,
				HashBlake2128:            2392870,
				HashBlake2128PerByte:     1563,
			},
		},
	}

	testGenesis.Genesis = Fields{
		Runtime: hrData,
	}

	// Grab json encoded bytes
	bz, err := json.Marshal(testGenesis)
	require.NoError(t, err)

	// Write to temp file
	filename := filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(filename, bz, os.ModePerm)
	require.NoError(t, err)

	// create genesis based on file just created, this will fill Raw field of genesis
	testGenesisProcessed, err := NewGenesisFromJSON(filename, 2)
	require.NoError(t, err)

	require.Equal(t, expectedGenesis.Genesis.Raw, testGenesisProcessed.Genesis.Raw)
}

func TestFormatKey(t *testing.T) {
	kv := &keyValue{
		key: []string{"babe", "Authorities"},
	}

	out, err := formatKey(kv)
	require.NoError(t, err)
	require.Equal(t, BABEAuthoritiesKeyHex, out)
}
