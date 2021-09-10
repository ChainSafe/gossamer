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
	expData := common.MustHexToBytes("0x0108d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01000000000000008eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a4801000000000000004d58630000000000000000000000000000000000000000000000000000000000")

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
