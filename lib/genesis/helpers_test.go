// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

func TestNewGenesisRawFromJSON(t *testing.T) {
	// Create temp file
	file, err := os.CreateTemp("", "genesis-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	testRaw := map[string]map[string]string{}
	testRaw["top"] = map[string]string{"0x3a636f6465": "0x0102"}

	expected := TestGenesis
	expected.Genesis = Fields{Raw: testRaw}

	// Grab json encoded bytes
	bz, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	// Write to temp file
	_, err = file.Write(bz)
	if err != nil {
		t.Fatal(err)
	}

	genesis, err := NewGenesisFromJSONRaw(file.Name())
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
	expRaw["top"]["0x3a636f6465"] = "0xfoo"
	expRaw["top"]["0x3a6772616e6470615f617574686f726974696573"] = "0x010834602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a6910000000000000000"                                                                                                                                                                                             // raw grandpa authorities
	expRaw["top"]["0x1cb6f36e027abb2091cfb5110ab5087f5e0621c4869aa60c02be9adcc98a0d1d"] = "0x08d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000"                                                                                                                                                                       // raw babe authorities
	expRaw["top"]["0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x0000000000000000000000007aeb9049000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" // raw system account
	expRaw["top"][common.BytesToHex(common.UpgradedToDualRefKey)] = "0x01"
	expRaw["top"]["0x426e15054d267946093858132eb537f1a47a9ff5cd5bf4d848a80a0b1a947dc3"] = "0x00000000000000000000000000000000"                                                                                                                 // Society
	expRaw["top"]["0x426e15054d267946093858132eb537f1ba7fb8745735dc3be2a2c61a72c39e78"] = "0x0101d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48"             // Society
	expRaw["top"]["0x426e15054d267946093858132eb537f1d0b4a3f7631f0c0e761898fe198211de"] = "0xe7030000"                                                                                                                                         // Society                                                                                                                                       // Staking
	expRaw["top"]["0x5f3e4907f716ac89b6347d15ececedca138e71612491192d68deab7e6f563fe1"] = "0x02000000"                                                                                                                                         // Staking
	expRaw["top"]["0x5f3e4907f716ac89b6347d15ececedca5579297f4dfb9609e7e4c2ebab9ce40a"] = "0x80be5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f"                                                                               // Staking
	expRaw["top"]["0x5f3e4907f716ac89b6347d15ececedcaac0a2cbf8e355f5ea6cb2de8727bfb0c"] = "0x54000000"                                                                                                                                         // Staking
	expRaw["top"]["0x5f3e4907f716ac89b6347d15ececedcab49a2738eeb30896aacb8b3fb46471bd"] = "0x01000000"                                                                                                                                         // Staking
	expRaw["top"]["0x5f3e4907f716ac89b6347d15ececedcac29a0310e1bb45d20cace77ccb62c97d"] = "0x00e1f505"                                                                                                                                         // Staking
	expRaw["top"]["0x5f3e4907f716ac89b6347d15ececedcaf7dad0317324aecae8744b87fc95f2f3"] = "0x00e1f505"                                                                                                                                         // Staking
	expRaw["top"]["0x5f3e4907f716ac89b6347d15ececedcaf7dad0317324aecae8744b87fc95f2f3"] = zeroByte                                                                                                                                             // Staking
	expRaw["top"]["0xcec5070d609dd3497f72bde07fc96ba04c014e6bf8b8c2c011e7290b85696bb3e535263148daaf49be5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f"] = "0xbe5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f" //Session
	expRaw["top"]["0xcec5070d609dd3497f72bde07fc96ba0726380404683fc89e8233450c8aa195066b8d48da86b869b6261626580d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session
	expRaw["top"]["0xcec5070d609dd3497f72bde07fc96ba0726380404683fc89e8233450c8aa1950c9b0c13125732d276175646980d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session
	expRaw["top"]["0xcec5070d609dd3497f72bde07fc96ba0726380404683fc89e8233450c8aa1950ed43a85541921049696d6f6e80d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session
	expRaw["top"]["0xcec5070d609dd3497f72bde07fc96ba0726380404683fc89e8233450c8aa1950f5537bdb2a1f626b6772616e8088dc3417d5058ec4b4503e0c12ea1a0a89be200fe98922423d4334014fa6b0ee"] = "0x"                                                       // Session
	expRaw["top"]["0x11f3ba2e1cdd6d62f2ff9b5589e7ff81ba7fb8745735dc3be2a2c61a72c39e78"] = zeroByte                                                                                                                                             // Instance1Collective
	expRaw["top"]["0x5f3e4907f716ac89b6347d15ececedca28dccb559b95c40168a1b2696581b5a7"] = "0x00000000000000000000000000000000"                                                                                                                 // Staking.CanceledSlashPayout
	expRaw["top"]["0x8985776095addd4789fccbce8ca77b23ba7fb8745735dc3be2a2c61a72c39e78"] = "0x08d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48"               // Instance2Collective
	expRaw["top"]["0x492a52699edf49c972c21db794cfcf57ba7fb8745735dc3be2a2c61a72c39e78"] = zeroByte                                                                                                                                             // Instance1Membership
	// Contract
	expRaw["top"]["0x4342193e496fab7ec59d615ed0dc5530d2d505c0e6f76fd7ce0796ebe187401c"] = "0x010000000001040000000002000000010000800000001000000000100000000100002000000000000800150600004c6b02004c8103009e1800000b1d0000160d0000651800006d2b00008a000000f5700100fdf602008e070000440600006e070000030600004f180000b528000091070000b60da70827080000590800006a0a0000ef070000560800004a0800008e080000f509000061090000dd090000a10a00009c090000e409000091090000650900001e0a0000120a0000ae09000099090000060a00006b2000000b1d000051200000221d000094090000ad090000b6090000160a0000660a0000fd090000260a0000440a0000d41a2a0000000000a0c729000000000092122900000000001ab5580000000000ba1c290000000000e000290000000000b0ef280000000000ee325c0000000000dec1280000000000ca07290000000000c07d4e00000000009c77140000000000303a7200000000000b01000000000000f0ab450000000000ff0200000000000060a21c270000000030078d31000000002635af09000000000ae164000000000038b18e0000000000b6b1cc0700000000890900000000000040036c00000000008ad6e21100000000de020000000000006e67cc080000000078f6110200000000e605000000000000acb1b50a00000000b24419090000000092579f08000000004702000000000000240300000000000016eaeb220000000055020000000000003503000000000000db0a000000000000f4802600000000006a100000000000006a9a280000000000220d0000000000004e9c2400000000001c0600000000000026832400000000001b06000000000000"
	expectedGenesis.Genesis = Fields{
		Raw: expRaw,
	}

	// Create temp file
	file, err := os.CreateTemp("", "genesis_hr-test")
	require.NoError(t, err)

	defer os.Remove(file.Name())

	// create human readable test genesis
	testGenesis := &Genesis{}
	hrData := make(map[string]map[string]interface{})
	hrData["System"] = map[string]interface{}{"code": "0xfoo"} // system code entry
	hrData["Babe"] = make(map[string]interface{})
	hrData["Babe"]["Authorities"] = []interface{}{"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY", 1} // babe authority data
	hrData["Grandpa"] = make(map[string]interface{})
	hrData["Grandpa"]["Authorities"] = []interface{}{"5DFNv4Txc4b88qHqQ6GG4D646QcT4fN3jjS2G3r1PyZkfDut", 0} // grandpa authority data
	hrData["Balances"] = make(map[string]interface{})
	hrData["Balances"]["balances"] = []interface{}{"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY", 1234234234} // balances
	// Add test cases for new fields...
	hrData["Society"] = make(map[string]interface{})
	hrData["Society"] = map[string]interface{}{
		"Pot":        0,
		"MaxMembers": 999,
		"Members": []interface{}{
			"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
			"5FHneW46xGXgs5mUiveU4sbTyGBzmstUspZC92UhjJM694ty",
		}}

	hrData["Staking"] = make(map[string]interface{})
	hrData["Staking"] = map[string]interface{}{
		"HistoryDepth":          84,
		"ValidatorCount":        2,
		"MinimumValidatorCount": 1,
		"ForceEra":              "NotForcing",
		"SlashRewardFraction":   100000000,
		"Invulnerables": []interface{}{
			"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
		},
		"CanceledSlashPayout": 0,
	}

	hrData["Session"] = make(map[string]interface{})
	hrData["Session"] = map[string]interface{}{
		"NextKeys": []interface{}{
			[]interface{}{
				"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
				"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
				map[string]interface{}{
					"grandpa":             "5FA9nQDVg267DEd8m1ZypXLBnvN7SFxYwV7ndqSYGiN9TTpu",
					"babe":                "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
					"im_online":           "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
					"authority_discovery": "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
				},
			},
		},
	}

	hrData["Instance1Collective"] = make(map[string]interface{})
	hrData["Instance1Collective"] = map[string]interface{}{
		"Phantom": nil,
		"Members": []interface{}{},
	}
	hrData["Instance2Collective"] = make(map[string]interface{})
	hrData["Instance2Collective"] = map[string]interface{}{
		"Phantom": nil,
		"Members": []interface{}{
			"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
			"5FHneW46xGXgs5mUiveU4sbTyGBzmstUspZC92UhjJM694ty",
		},
	}

	hrData["Instance1Membership"] = make(map[string]interface{})
	hrData["Instance1Membership"] = map[string]interface{}{
		"Members": []interface{}{},
		"Phantom": nil,
	}

	hrData["Contracts"] = make(map[string]interface{})
	hrData["Contracts"] = map[string]interface{}{
		"CurrentSchedule": map[string]interface{}{
			"version":        0,
			"enable_println": true,
			"limits": map[string]interface{}{
				"event_topics":  4,
				"stack_height":  512,
				"globals":       256,
				"parameters":    128,
				"memory_pages":  16,
				"table_size":    4096,
				"br_table_size": 256,
				"subject_len":   32,
				"code_size":     524288,
			},
			"instruction_weights": map[string]interface{}{
				"i64const":                1557,
				"i64load":                 158540,
				"i64store":                229708,
				"select":                  6302,
				"if":                      7435,
				"br":                      3350,
				"br_if":                   6245,
				"br_table":                11117,
				"br_table_per_entry":      138,
				"call":                    94453,
				"call_indirect":           194301,
				"call_indirect_per_param": 1934,
				"local_get":               1604,
				"local_set":               1902,
				"local_tee":               1539,
				"global_get":              6223,
				"global_set":              10421,
				"memory_current":          1937,
				"memory_grow":             145165750,
				"i64clz":                  2087,
				"i64ctz":                  2137,
				"i64popcnt":               2666,
				"i64eqz":                  2031,
				"i64extendsi32":           2134,
				"i64extendui32":           2122,
				"i32wrapi64":              2190,
				"i64eq":                   2549,
				"i64ne":                   2401,
				"i64lts":                  2525,
				"i64ltu":                  2721,
				"i64gts":                  2460,
				"i64gtu":                  2532,
				"i64les":                  2449,
				"i64leu":                  2405,
				"i64ges":                  2590,
				"i64geu":                  2578,
				"i64add":                  2478,
				"i64sub":                  2457,
				"i64mul":                  2566,
				"i64divs":                 8299,
				"i64divu":                 7435,
				"i64rems":                 8273,
				"i64remu":                 7458,
				"i64and":                  2452,
				"i64or":                   2477,
				"i64xor":                  2486,
				"i64shl":                  2582,
				"i64shrs":                 2662,
				"i64shru":                 2557,
				"i64rotl":                 2598,
				"i64rotr":                 2628,
			},
			"host_fn_weights": map[string]interface{}{
				"caller":                      2759380,
				"address":                     2738080,
				"gas_left":                    2691730,
				"balance":                     5813530,
				"value_transferred":           2694330,
				"minimum_balance":             2687200,
				"tombstone_deposit":           2682800,
				"rent_allowance":              6042350,
				"block_number":                2671070,
				"now":                         2688970,
				"weight_to_fee":               5144000,
				"gas":                         1341340,
				"input":                       7486000,
				"input_per_byte":              267,
				"return":                      4566000,
				"return_per_byte":             767,
				"terminate":                   656188000,
				"restore_to":                  831326000,
				"restore_to_per_delta":        162477350,
				"random":                      6611210,
				"deposit_event":               9351480,
				"deposit_event_per_topic":     130855350,
				"deposit_event_per_byte":      2441,
				"set_rent_allowance":          7078720,
				"set_storage":                 300078730,
				"set_storage_per_byte":        734,
				"clear_storage":               147613550,
				"get_storage":                 34731640,
				"get_storage_per_byte":        1510,
				"transfer":                    179679660,
				"call":                        152650930,
				"call_transfer_surcharge":     144660370,
				"call_per_input_byte":         583,
				"call_per_output_byte":        804,
				"instantiate":                 585886230,
				"instantiate_per_input_byte":  597,
				"instantiate_per_output_byte": 821,
				"instantiate_per_salt_byte":   2779,
				"hash_sha2_256":               2523380,
				"hash_sha2_256_per_byte":      4202,
				"hash_keccak_256":             2660970,
				"hash_keccak_256_per_byte":    3362,
				"hash_blake2_256":             2399310,
				"hash_blake2_256_per_byte":    1564,
				"hash_blake2_128":             2392870,
				"hash_blake2_128_per_byte":    1563,
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
	_, err = file.Write(bz)
	require.NoError(t, err)

	// create genesis based on file just created, this will fill Raw field of genesis
	testGenesisProcessed, err := NewGenesisFromJSON(file.Name(), 2)
	require.NoError(t, err)

	require.Equal(t, expectedGenesis.Genesis.Raw, testGenesisProcessed.Genesis.Raw)
}

func TestFormatKey(t *testing.T) {
	kv := &keyValue{
		key: []string{"Babe", "Authorities"},
	}

	out, err := formatKey(kv)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("0x%x", runtime.BABEAuthoritiesKey()), out)
}

func TestNewTrieFromGenesis(t *testing.T) {
	var rawGenesis = &Genesis{}
	raw := make(map[string]map[string]string)
	raw["top"] = make(map[string]string)
	raw["top"]["0x3a636f6465"] = "0x0102" // raw :code
	rawGenesis.Genesis = Fields{
		Raw: raw,
	}

	expTrie := trie.NewEmptyTrie()
	expTrie.Put([]byte(`:code`), []byte{1, 2})

	trie, err := NewTrieFromGenesis(rawGenesis)
	require.NoError(t, err)

	require.Equal(t, expTrie, trie)
}
