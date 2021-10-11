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

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestBabePrimaryPreDigest_EncodeAndDecode(t *testing.T) {
	bh := NewBabeDigest()
	err := bh.Set(BabePrimaryPreDigest{
		VRFOutput:      [sr25519.VRFOutputLength]byte{0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152, 85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40, 246, 76, 38, 235, 204, 43, 31, 179, 28},
		VRFProof:       [sr25519.VRFProofLength]byte{120, 23, 235, 159, 115, 122, 207, 206, 123, 232, 75, 243, 115, 255, 131, 181, 219, 241, 200, 206, 21, 22, 238, 16, 68, 49, 86, 99, 76, 139, 39, 0, 102, 106, 181, 136, 97, 141, 187, 1, 234, 183, 241, 28, 27, 229, 133, 8, 32, 246, 245, 206, 199, 142, 134, 124, 226, 217, 95, 30, 176, 246, 5, 3},
		AuthorityIndex: 17,
		SlotNumber:     420,
	})
	require.NoError(t, err)

	enc, err := scale.Marshal(bh)
	require.NoError(t, err)
	bh2, err := DecodeBabePreDigest(enc)
	require.NoError(t, err)
	require.Equal(t, bh.Value(), bh2)
}

func TestBabeSecondaryPlainPreDigest_EncodeAndDecode(t *testing.T) {
	bh := NewBabeDigest()
	err := bh.Set(BabeSecondaryPlainPreDigest{
		AuthorityIndex: 17,
		SlotNumber:     420,
	})
	require.NoError(t, err)

	enc, err := scale.Marshal(bh)
	require.NoError(t, err)
	bh2, err := DecodeBabePreDigest(enc)
	require.NoError(t, err)
	require.Equal(t, bh.Value(), bh2)
}

func TestBabeSecondaryVRFPreDigest_EncodeAndDecode(t *testing.T) {
	bh := NewBabeDigest()
	err := bh.Set(BabeSecondaryVRFPreDigest{
		VrfOutput:      [sr25519.VRFOutputLength]byte{0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152, 85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40, 246, 76, 38, 235, 204, 43, 31, 179, 28},
		VrfProof:       [sr25519.VRFProofLength]byte{120, 23, 235, 159, 115, 122, 207, 206, 123, 232, 75, 243, 115, 255, 131, 181, 219, 241, 200, 206, 21, 22, 238, 16, 68, 49, 86, 99, 76, 139, 39, 0, 102, 106, 181, 136, 97, 141, 187, 1, 234, 183, 241, 28, 27, 229, 133, 8, 32, 246, 245, 206, 199, 142, 134, 124, 226, 217, 95, 30, 176, 246, 5, 3},
		AuthorityIndex: 17,
		SlotNumber:     420,
	})
	require.NoError(t, err)

	enc, err := scale.Marshal(bh)
	require.NoError(t, err)
	bh2, err := DecodeBabePreDigest(enc)
	require.NoError(t, err)
	require.Equal(t, bh.Value(), bh2)
}
