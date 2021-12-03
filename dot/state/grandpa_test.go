// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"

	"github.com/stretchr/testify/require"
)

var (
	kr, _     = keystore.NewEd25519Keyring()
	testAuths = []types.GrandpaVoter{
		{Key: *kr.Alice().Public().(*ed25519.PublicKey), ID: 0},
	}
)

func TestNewGrandpaStateFromGenesis(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, testAuths)
	require.NoError(t, err)

	currSetID, err := gs.GetCurrentSetID()
	require.NoError(t, err)
	require.Equal(t, genesisSetID, currSetID)

	auths, err := gs.GetAuthorities(currSetID)
	require.NoError(t, err)
	require.Equal(t, testAuths, auths)

	num, err := gs.GetSetIDChange(0)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(0), num)
}

func TestGrandpaState_SetNextChange(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, testAuths)
	require.NoError(t, err)

	err = gs.SetNextChange(testAuths, big.NewInt(1))
	require.NoError(t, err)

	auths, err := gs.GetAuthorities(genesisSetID + 1)
	require.NoError(t, err)
	require.Equal(t, testAuths, auths)

	atBlock, err := gs.GetSetIDChange(genesisSetID + 1)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1), atBlock)
}

func TestGrandpaState_IncrementSetID(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, testAuths)
	require.NoError(t, err)

	err = gs.IncrementSetID()
	require.NoError(t, err)

	setID, err := gs.GetCurrentSetID()
	require.NoError(t, err)
	require.Equal(t, genesisSetID+1, setID)
}

func TestGrandpaState_GetSetIDByBlockNumber(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, testAuths)
	require.NoError(t, err)

	err = gs.SetNextChange(testAuths, big.NewInt(100))
	require.NoError(t, err)

	setID, err := gs.GetSetIDByBlockNumber(big.NewInt(50))
	require.NoError(t, err)
	require.Equal(t, genesisSetID, setID)

	setID, err = gs.GetSetIDByBlockNumber(big.NewInt(100))
	require.NoError(t, err)
	require.Equal(t, genesisSetID, setID)

	setID, err = gs.GetSetIDByBlockNumber(big.NewInt(101))
	require.NoError(t, err)
	require.Equal(t, genesisSetID+1, setID)

	err = gs.IncrementSetID()
	require.NoError(t, err)

	setID, err = gs.GetSetIDByBlockNumber(big.NewInt(100))
	require.NoError(t, err)
	require.Equal(t, genesisSetID, setID)

	setID, err = gs.GetSetIDByBlockNumber(big.NewInt(101))
	require.NoError(t, err)
	require.Equal(t, genesisSetID+1, setID)
}

func TestGrandpaState_LatestRound(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, testAuths)
	require.NoError(t, err)

	r, err := gs.GetLatestRound()
	require.NoError(t, err)
	require.Equal(t, uint64(0), r)

	err = gs.SetLatestRound(99)
	require.NoError(t, err)

	r, err = gs.GetLatestRound()
	require.NoError(t, err)
	require.Equal(t, uint64(99), r)
}
