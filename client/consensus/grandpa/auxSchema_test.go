// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
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
	return nil
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

func TestDecodeFromV0MigratesDataFormat(t *testing.T) {
	client := newDummyClient(t)

	kpA, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	var authorities AuthorityList
	authorities = append(authorities, Authority{
		Key:    kpA.Public(),
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
			currentAuthorities: authorities,
			pendingChanges:     []V0PendingChange[Hash, uint]{},
			setID:              setId,
		}

		voterSetState := roundInfo[Hash, uint]{
			roundNumber: uint64(roundNumber),
			roundState:  roundState,
		}

		// I don't think our scale can support this
		insert := map[string][]byte{}
		insert[string(AUTHORITY_SET_KEY)] = scale.MustMarshal(authoritySet)
		insert[string(SET_STATE_KEY)] = scale.MustMarshal(voterSetState)

		fmt.Println("here")
		err := client.InsertAux(insert, nil)
		require.NoError(t, err)

		fmt.Println(client)
	}

}
