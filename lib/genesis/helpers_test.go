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
	const oneByte = "0x01"

	expRaw := make(map[string]map[string]string)
	expRaw["top"] = make(map[string]string)
	expRaw["top"]["0x3a636f6465"] = "0xfoo"                                                                                                                                                                                                                                                                                                                    // raw system code
	expRaw["top"]["0x3a6772616e6470615f617574686f726974696573"] = "0x010834602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a6910000000000000000"                                                                                                                                                                                                     // raw grandpa authorities
	expRaw["top"]["0x014f204c006a2837deb5551ba5211d6ce887d1f35708af762efe7b709b5eff15"] = "0x08d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000"                                                                                                                                                                               // raw babe authorities
	expRaw["top"]["0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x000000000000000000000000000000007aeb9049000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" // Balances
	expRaw["top"][common.BytesToHex(common.UpgradedToDualRefKey)] = oneByte
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707138e71612491192d68deab7e6f563fe1"] = "0x02000000"                                                                                                                                         // Staking.ValidatorCount
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e7075579297f4dfb9609e7e4c2ebab9ce40a"] = "0x80be5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f"                                                                               // Staking.Invulnerables
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707b49a2738eeb30896aacb8b3fb46471bd"] = "0x01000000"                                                                                                                                         // Staking.MinimumValidatorCount
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707c29a0310e1bb45d20cace77ccb62c97d"] = "0x00e1f505"                                                                                                                                         // Staking.SlashRewardFraction
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707d32c6475a1afd11c5d3645883a408350"] = "0x00000000000000000000000000000000"                                                                                                                 // Staking.CanceledPayout
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707f7dad0317324aecae8744b87fc95f2f3"] = zeroByte                                                                                                                                             // Staking.ForceEra
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707666fdcbb473985b3ac933d13f4acff8d"] = "0x00000000"                                                                                                                                         // Staking.MinValidatorBond
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707cfcf8606ab1b2ac8c58f68f2551112be"] = "0x00"                                                                                                                                               // Staking.MaxValidatorCount
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707ed441ceb81326c56263efbb60c95c2e4"] = "0x00000000"                                                                                                                                         // Staking.MinNominatorBond
	expRaw["top"]["0x7bbd1b4c54319a153cc9fdbd5792e707fda3863edefdd0f36f86ab168187e2c7"] = "0x00"                                                                                                                                               // Staking.MaxNominatorCount
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b54c014e6bf8b8c2c011e7290b85696bb3e535263148daaf49be5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f"] = "0xbe5ddb1579b72e84524fc29e78609e3caf42e85aa118ebfe0b0ad404b5bdd25f" // Session
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa1950f5537bdb2a1f626b6772616e8088dc3417d5058ec4b4503e0c12ea1a0a89be200fe98922423d4334014fa6b0ee"] = "0x"                                                       // Session.Grandpa
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa195066b8d48da86b869b6261626580d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session.Babe
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa1950ed43a85541921049696d6f6e80d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session.ImOnline
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa195079b38849014a07307061726180d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session.ParaValidator
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa19504a8e42157609c6c86173676e80d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session.ParaAssignment
	expRaw["top"]["0x3113eb570b4ee7c041d467c912beb8b5726380404683fc89e8233450c8aa1950c9b0c13125732d276175646980d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x"                                                       // Session.AuthorityDiscovery

	expRaw["top"]["0x26aa394eea5630e07c48ae0c9558cef7c21aab032aaa6e946ca50ad39ab66603"] = oneByte
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

	hrData.Staking = &staking{
		ValidatorCount:        2,
		MinimumValidatorCount: 1,
		Invulnerables: []string{
			"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
		},
		ForceEra:            "NotForcing",
		SlashRewardFraction: 100000000,
		CanceledPayout:      zeroOfUint128,
		MinNominatorBond:    0,
		MinValidatorBond:    0,
		MaxValidatorCount:   nil,
		MaxNominatorCount:   nil,
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
					ParaValidator:      "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
					ParaAssignment:     "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
					AuthorityDiscovery: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
				},
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
		key: []string{"Babe", "Authorities"},
	}

	out, err := formatKey(kv)
	require.NoError(t, err)
	require.Equal(t, BABEAuthoritiesKeyHex, out)
}
