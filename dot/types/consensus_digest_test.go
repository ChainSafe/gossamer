// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestBabeEncodeAndDecode(t *testing.T) {
	expData := common.MustHexToBytes("0x0108d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01000000000000008eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a4801000000000000004d58630000000000000000000000000000000000000000000000000000000000") //nolint:lll

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	authA := AuthorityRaw{
		Key:    keyring.Alice().Public().(*sr25519.PublicKey).AsBytes(),
		Weight: 1,
	}

	authB := AuthorityRaw{
		Key:    keyring.Bob().Public().(*sr25519.PublicKey).AsBytes(),
		Weight: 1,
	}

	var d = NewBabeConsensusDigest()
	err = d.Set(NextEpochData{
		Authorities: []AuthorityRaw{authA, authB},
		Randomness:  [32]byte{77, 88, 99},
	})
	require.NoError(t, err)

	enc, err := scale.Marshal(d)
	require.NoError(t, err)
	require.Equal(t, expData, enc)

	var dec = NewBabeConsensusDigest()
	err = scale.Unmarshal(enc, &dec)
	require.NoError(t, err)
	require.Equal(t, d, dec)
}

func TestBabeDecodeVersionedNextConfigData(t *testing.T) {
	// Block #5608275 NextConfigData digest
	enc := common.MustHexToBytes("0x03010100000000000000040000000000000002")

	var dec = NewBabeConsensusDigest()
	err := scale.Unmarshal(enc, &dec)
	require.NoError(t, err)

	decValue, err := dec.Value()
	require.NoError(t, err)

	nextVersionedConfigData := decValue.(VersionedNextConfigData)

	nextConfigData, err := nextVersionedConfigData.Value()
	require.NoError(t, err)

	nextConfigDataV1 := nextConfigData.(NextConfigDataV1)

	require.GreaterOrEqual(t, 1, int(nextConfigDataV1.C1))
	require.GreaterOrEqual(t, 4, int(nextConfigDataV1.C2))
	require.GreaterOrEqual(t, 2, int(nextConfigDataV1.SecondarySlots))
}
