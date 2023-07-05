// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/minio/sha256-simd"
	"github.com/stretchr/testify/require"
)

func createHeader(t *testing.T, n uint) (header *types.Header) {
	t.Helper()

	randomBytes := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, randomBytes)
	require.NoError(t, err)

	hasher := sha256.New()
	_, err = hasher.Write(randomBytes)
	require.NoError(t, err)

	header = types.NewEmptyHeader()
	header.Number = n

	// so that different headers for the same number get different hashes
	header.ParentHash = common.NewHash(hasher.Sum(nil))

	header.Hash()
	return header
}

func checkSlotToMapKeyExists(t *testing.T, db chaindb.Database, slotNumber uint64) bool {
	t.Helper()

	slotEncoded := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotEncoded, slotNumber)

	slotToHeaderKey := bytes.Join([][]byte{slotHeaderMapKey, slotEncoded[:]}, nil)

	_, err := db.Get(slotToHeaderKey)
	if err != nil {
		if errors.Is(err, chaindb.ErrKeyNotFound) {
			return false
		}

		t.Fatalf("unexpected error while getting key: %s", err)
	}

	return true
}

func Test_checkEquivocation(t *testing.T) {
	inMemoryDB, err := chaindb.NewBadgerDB(&chaindb.Config{
		DataDir:  t.TempDir(),
		InMemory: true,
	})
	require.NoError(t, err)

	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	alicePublicKey := kr.KeyAlice.Public().(*sr25519.PublicKey)
	aliceAuthorityID := types.AuthorityID(alicePublicKey.AsBytes())

	header1 := createHeader(t, 1) // @ slot 2
	header2 := createHeader(t, 2) // @ slot 2
	header3 := createHeader(t, 2) // @ slot 4
	header4 := createHeader(t, 3) // @ slot MAX_SLOT_CAPACITY + 4
	header5 := createHeader(t, 4) // @ slot MAX_SLOT_CAPACITY + 4
	header6 := createHeader(t, 3) // @ slot 4

	slotState := NewSlotState(inMemoryDB)

	// It's ok to sign same headers.
	equivProf, err := slotState.CheckEquivocation(2, 2, header1, aliceAuthorityID)
	require.NoError(t, err)
	require.Nil(t, equivProf)

	equivProf, err = slotState.CheckEquivocation(3, 2, header1, aliceAuthorityID)
	require.NoError(t, err)
	require.Nil(t, equivProf)

	// But not two different headers at the same slot.
	equivProf, err = slotState.CheckEquivocation(4, 2, header2, aliceAuthorityID)
	require.NoError(t, err)
	require.NotNil(t, equivProf)
	require.Equal(t, &types.BabeEquivocationProof{
		Slot:         2,
		Offender:     aliceAuthorityID,
		FirstHeader:  *header1,
		SecondHeader: *header2,
	}, equivProf)

	// Different slot is ok.
	equivProf, err = slotState.CheckEquivocation(5, 4, header3, aliceAuthorityID)
	require.NoError(t, err)
	require.Nil(t, equivProf)

	// Here we trigger pruning and save header 4.
	equivProf, err = slotState.CheckEquivocation(
		pruningBound+2, maxSlotCapacity+4, header4, aliceAuthorityID)
	require.NoError(t, err)
	require.Nil(t, equivProf)

	require.False(t, checkSlotToMapKeyExists(t, slotState.db, 2))
	require.False(t, checkSlotToMapKeyExists(t, slotState.db, 4))

	// This fails because header 5 is an equivocation of header 4.
	equivProf, err = slotState.CheckEquivocation(
		pruningBound+3, maxSlotCapacity+4, header5, aliceAuthorityID)
	require.NoError(t, err)
	require.NotNil(t, equivProf)

	require.Equal(t, &types.BabeEquivocationProof{
		Slot:         maxSlotCapacity + 4,
		Offender:     aliceAuthorityID,
		FirstHeader:  *header4,
		SecondHeader: *header5,
	}, equivProf)

	// This is ok because we pruned the corresponding header. Shows that we are pruning.
	equivProf, err = slotState.CheckEquivocation(
		pruningBound+4, 4, header6, aliceAuthorityID)
	require.NoError(t, err)
	require.Nil(t, equivProf)
}
