// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package dot

import (
	"encoding/binary"
	"fmt"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/OneOfOne/xxhash"
	"github.com/btcsuite/btcutil/base58"
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// TestNewConfig tests the NewTestConfig method
func TestNewConfig(t *testing.T) {
	cfg := NewTestConfig(t)

	defer utils.RemoveTestDir(t)

	// TODO: improve dot tests #687
	require.NotNil(t, cfg)
}

// TestNewConfigAndFile tests the NewTestConfigWithFile method
func TestNewConfigAndFile(t *testing.T) {
	testCfg, testCfgFile := NewTestConfigWithFile(t)

	defer utils.RemoveTestDir(t)

	// TODO: improve dot tests #687
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)
}

// TestInitNode
func TestNewTestGenesis(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()
	fmt.Printf("FileName %v\n", genFile.Name())
}

func TestNewTestGenesisFromJSONHR(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONHR("../gossamer_genesis.json")
	//gen, err := genesis.NewGenesisFromJSONHR("../myCustomSpec.json")
	require.NoError(t, err)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	fmt.Printf("alice address %v\n", kr.Alice.Public().Address())
	fmt.Printf("alice hex %x\n", kr.Alice.Public().Encode())
	//npk, err := ed25519.NewPublicKey(kr.Alice.Public().Encode())
	//require.NoError(t, err)
	//fmt.Printf("npk %v\n", npk.Address())
	fmt.Printf("gen %v\n", gen.Name)
	//pubB1 := common.MustHexToBytes("0x26aa394eea5630e07c48ae0c9558cef702a5c1b19ab7a04f536c519aca4983ac")
	pubB1 := common.MustHexToBytes("0x1206960f920a23f7f4c43cc9081ec2ed0721f31a9bef2c10fd7602e16e08a32c")
	pk1, err := ed25519.NewPublicKey(pubB1)
	fmt.Printf("pk1 adderess %v\n", pk1.Address())
	pk1Check := base58.Decode(string(pk1.Address()))
	fmt.Printf("pk1check %v\n", pk1Check)
	fmt.Printf("pk1check %x\n", pk1Check[1:33])
}

func TestHash(t *testing.T) {
	//logger.Trace("[ext_twox_128] executing...")
	//instanceContext := wasm.IntoInstanceContext(context)
	//memory := instanceContext.Memory().Data()
	//memory := []byte(`Babe Authorities`)  // 0x886726f904d8372fdabb7707870c2fad
	memory := []byte(`System Number`)  // 0x8cb577756012d928f17362e0741f9f2c
	logger.Trace("[ext_twox_128] hashing...", "value", fmt.Sprintf("%s", memory[:]))

	// compute xxHash64 twice with seeds 0 and 1 applied on given byte array
	h0 := xxhash.NewS64(0) // create xxHash with 0 seed
	_, err := h0.Write(memory[0 : len(memory)])
	if err != nil {
		logger.Error("[ext_twox_128]", "error", err)
		return
	}
	res0 := h0.Sum64()
	hash0 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash0, res0)

	h1 := xxhash.NewS64(1) // create xxHash with 1 seed
	_, err = h1.Write(memory[0 : len(memory)])
	if err != nil {
		logger.Error("[ext_twox_128]", "error", err)
		return
	}
	res1 := h1.Sum64()
	hash1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash1, res1)

	//concatenated result
	both := append(hash0, hash1...)
	fmt.Printf("both: %x\n", both)
}