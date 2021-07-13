// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package grandpa

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestPubkeyToVoter(t *testing.T) {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	state := NewState(voters, 0, 0)
	voter, err := state.pubkeyToVoter(kr.Alice().Public().(*ed25519.PublicKey))
	require.NoError(t, err)
	require.Equal(t, voters[0], voter)
}

func TestSignedVoteEncoding(t *testing.T) {
	just := &SignedVote{
		Vote:        testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}

	enc, err := just.Encode()
	require.NoError(t, err)

	rw := &bytes.Buffer{}
	rw.Write(enc)
	dec := new(SignedVote)
	_, err = dec.Decode(rw)
	require.NoError(t, err)
	require.Equal(t, just, dec)
}

func TestSignedVoteArrayEncoding(t *testing.T) {
	just := []*SignedVote{
		{
			Vote:        testVote,
			Signature:   testSignature,
			AuthorityID: testAuthorityID,
		},
	}

	enc, err := scale.Marshal(just)
	require.NoError(t, err)

	var dec []*SignedVote
	err = scale.Unmarshal(enc, &dec)
	require.NoError(t, err)
	require.Equal(t, just, dec)
}

func TestJustification(t *testing.T) {
	just := &SignedVote{
		Vote:        testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}

	fj := &Justification{
		Round: 99,
		Commit: &Commit{
			Precommits: []*SignedVote{just},
		},
	}
	enc, err := scale.Marshal(fj)
	require.NoError(t, err)

	var dec *Justification
	err = scale.Unmarshal(enc, &dec)
	require.NoError(t, err)
	require.Equal(t, fj, dec)
}

func TestJustification_Decode(t *testing.T) {
	// data received from network
	data := common.MustHexToBytes("0x01020000000000000001700e648a80bf01944ca2a5ae4da4fea86810d02b549d1e399c06eee938b973f102000000080101700e648a80bf01944ca2a5ae4da4fea86810d02b549d1e399c06eee938b973f10200000089aea3ec1b522a15dd7c644ed60d332e0da76b761f2a8e00a90cf3ef2052511399bb06a8217952b6fa63119b7ccdb1da498aaa42eb80fef02c6c77b9fb0aec0f34602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a6910101700e648a80bf01944ca2a5ae4da4fea86810d02b549d1e399c06eee938b973f1020000009a406003a1d551d5245e7e6a16d497ed3fd45f63b80402a83d2d23694f4e298ac6982f02aefcfd65487c52dcbcf88789e52b99da7d4effb9cc319ad0101d4e0c94a297125bf31bc15e2a2f1d7d44d2c2a99ce3ed81fdc3a7acf4a4cc30480fb7")

	var fj *Justification
	err := scale.Unmarshal(data, &fj)
	require.NoError(t, err)
	require.Equal(t, uint64(2), fj.Round)
	require.Equal(t, uint32(2), fj.Commit.Number)
	require.Equal(t, common.MustHexToHash("0x700e648a80bf01944ca2a5ae4da4fea86810d02b549d1e399c06eee938b973f1"), fj.Commit.Hash)
	require.Equal(t, 2, len(fj.Commit.Precommits))
}
