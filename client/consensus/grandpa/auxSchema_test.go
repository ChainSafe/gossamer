// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"testing"
)

type dummyClient map[string][]byte

func (client dummyClient) InsertAux(insert map[string][]byte, deleted []string) error {
	for key, val := range insert {
		client[key] = val
	}

	for _, key := range deleted {
		delete(client, key)
	}

	return nil
}

func (client dummyClient) GetAux(key []byte) *[]byte {
	res := client[string(key)]
	if len(res) == 0 {
		return nil
	}
	return &res
}

func newDummyClient(t *testing.T) dummyClient {
	t.Helper()
	return dummyClient{}
}

func TestDummyClientInsert(t *testing.T) {
	client := newDummyClient(t)

	insert := map[string][]byte{}
	insert["a"] = []byte{0}
	insert["b"] = []byte{1}
	insert["c"] = []byte{2}

	err := client.InsertAux(insert, nil)
	require.NoError(t, err)

	insertNew := map[string][]byte{}
	insertNew["d"] = []byte{3}

	deleted := []string{"b"}

	err = client.InsertAux(insertNew, deleted)
	require.NoError(t, err)

	require.Equal(t, []byte{0}, client["a"])
	require.Nil(t, client["b"])
	require.Equal(t, []byte{2}, client["c"])
	require.Equal(t, []byte{3}, client["d"])
	require.Equal(t, 3, len(client))
}

func TestDecode(t *testing.T) {
	// Panics
	//kpA, err := ed25519.GenerateKeypair()
	//require.NoError(t, err)
	//
	//var authorities AuthorityList
	//authorities = append(authorities, Authority{
	//	Key:    kpA.Public(),
	//	Weight: 100,
	//})
	//
	//enc, err := scale.Marshal(&authorities)
	//require.NoError(t, err)
	//
	//var newAuth *AuthorityList
	//err = scale.Unmarshal(enc, &newAuth)
	//require.NoError(t, err)
	//require.Equal(t, authorities, newAuth)

	//kpA, err := ed25519.GenerateKeypair()
	//require.NoError(t, err)

	pubKey, err := ed25519.NewPublicKey([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	require.NoError(t, err)

	//pubKey := kpA.Public()

	enc, err := scale.Marshal(*pubKey)
	require.NoError(t, err)

	var auth ed25519.PublicKey
	err = scale.Unmarshal(enc, &auth)
	require.NoError(t, err)
	require.Equal(t, *pubKey, auth)
}

func TestDecodeFromV0MigratesDataFormat(t *testing.T) {
	client := newDummyClient(t)

	pubKey, err := ed25519.NewPublicKey([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	require.NoError(t, err)

	var authorities AuthorityList
	authorities = append(authorities, Authority{
		Key:    *pubKey,
		Weight: 100,
	})
	setId := uint64(3)
	roundNumber := 42
	roundState := finalityGrandpa.RoundState[Hash, uint]{
		PrevoteGHOST: &finalityGrandpa.HashNumber[Hash, uint]{
			Hash:   Hash{0},
			Number: 32,
		},
	}

	// they have block here, idk why
	{
		authoritySet := V0AuthoritySet[Hash, uint]{
			CurrentAuthorities: authorities,
			SetID:              setId,
			PendingChanges:     []V0PendingChange[Hash, uint]{},
		}

		voterSetState := roundInfo[Hash, uint]{
			roundNumber: uint64(roundNumber),
			roundState:  roundState,
		}

		insert := map[string][]byte{}
		insert[string(AUTHORITY_SET_KEY)] = scale.MustMarshal(authoritySet)
		insert[string(SET_STATE_KEY)] = scale.MustMarshal(voterSetState)

		err := client.InsertAux(insert, nil)
		require.NoError(t, err)
	}

	res := loadDecode(client, VERSION_KEY)
	require.Nil(t, res)

	// should perform the migration
	_, err = loadPersistent[Hash, uint](client, Hash{}, 0, func() (AuthorityList, error) {
		panic("error")
	})

	res = loadDecode(client, VERSION_KEY)
	require.NotNil(t, res)

	var version uint32
	err = scale.Unmarshal(*res, &version)
	require.NoError(t, err)
	require.Equal(t, CURRENT_VERSION, version)
}
