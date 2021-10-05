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
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"
	scale2 "github.com/ChainSafe/gossamer/pkg/scale"
)

// GrandpaAuthoritiesRaw represents a GRANDPA authority where their key is a byte array
type GrandpaAuthoritiesRaw struct {
	Key [ed25519.PublicKeyLength]byte
	ID  uint64
}

// Decode will decode the Reader into a GrandpaAuthoritiesRaw
func (a *GrandpaAuthoritiesRaw) Decode(r io.Reader) (*GrandpaAuthoritiesRaw, error) {
	key, err := common.Read32Bytes(r)
	if err != nil {
		return nil, err
	}

	id, err := common.ReadUint64(r)
	if err != nil {
		return nil, err
	}

	a = new(GrandpaAuthoritiesRaw)
	a.Key = key
	a.ID = id

	return a, nil
}

// FromRawEd25519 sets the Authority given GrandpaAuthoritiesRaw. It converts the byte representations of
// the authority public keys into a ed25519.PublicKey.
func (a *Authority) FromRawEd25519(raw GrandpaAuthoritiesRaw) error {
	key, err := ed25519.NewPublicKey(raw.Key[:])
	if err != nil {
		return err
	}

	a.Key = key
	a.Weight = raw.ID
	return nil
}

// GrandpaAuthoritiesRawToAuthorities turns a slice of GrandpaAuthoritiesRaw into a slice of Authority
func GrandpaAuthoritiesRawToAuthorities(adr []GrandpaAuthoritiesRaw) ([]Authority, error) {
	ad := make([]Authority, len(adr))
	for i, r := range adr {
		ad[i] = Authority{}
		err := ad[i].FromRawEd25519(r)
		if err != nil {
			return nil, err
		}
	}

	return ad, nil
}

// GrandpaVoter represents a GRANDPA voter
type GrandpaVoter struct {
	Key ed25519.PublicKey
	ID  uint64
}

// PublicKeyBytes returns the voter key as PublicKeyBytes
func (gv *GrandpaVoter) PublicKeyBytes() ed25519.PublicKeyBytes {
	return gv.Key.AsBytes()
}

// String returns a formatted GrandpaVoter string
func (gv *GrandpaVoter) String() string {
	return fmt.Sprintf("[key=0x%s id=%d]", gv.PublicKeyBytes(), gv.ID)
}

// Decode will decode the Reader into a GrandpaVoter
func (gv *GrandpaVoter) Decode(r io.Reader) error {
	keyBytes, err := common.Read32Bytes(r)
	if err != nil {
		return err
	}

	key, err := ed25519.NewPublicKey(keyBytes[:])
	if err != nil {
		return err
	}

	id, err := common.ReadUint64(r)
	if err != nil {
		return err
	}

	gv.Key = *key
	gv.ID = id
	return nil
}

// Encode will encode the GrandpaVoter
func (gv *GrandpaVoter) Encode() ([]byte, error) {
	enc := []byte{}
	b := gv.Key.Encode()
	enc = append(enc, b...)
	e, err := scale2.Marshal(gv.ID)
	if err != nil {
		return nil, err
	}
	enc = append(enc, e...)
	return enc, nil
}

// NewGrandpaVotersFromAuthorities returns an array of GrandpaVoters given an array of GrandpaAuthorities
func NewGrandpaVotersFromAuthorities(ad []Authority) []GrandpaVoter {
	v := make([]GrandpaVoter, len(ad))

	for i, d := range ad {
		if pk, ok := d.Key.(*ed25519.PublicKey); ok {
			v[i] = GrandpaVoter{
				Key: *pk,
				ID:  d.Weight,
			}
		}
	}

	return v
}

// NewGrandpaVotersFromAuthoritiesRaw returns an array of GrandpaVoters given an array of GrandpaAuthoritiesRaw
func NewGrandpaVotersFromAuthoritiesRaw(ad []GrandpaAuthoritiesRaw) ([]GrandpaVoter, error) {
	v := make([]GrandpaVoter, len(ad))

	for i, d := range ad {
		key, err := ed25519.NewPublicKey(d.Key[:])
		if err != nil {
			return nil, err
		}

		v[i] = GrandpaVoter{
			Key: *key,
			ID:  d.ID,
		}
	}

	return v, nil
}

// GrandpaVoters represents []*GrandpaVoter
type GrandpaVoters []GrandpaVoter

// String returns a formatted Voters string
func (v GrandpaVoters) String() string {
	str := ""
	for _, w := range v {
		str = str + w.String() + " "
	}
	return str
}

// EncodeGrandpaVoters returns an encoded GrandpaVoters
func EncodeGrandpaVoters(voters GrandpaVoters) ([]byte, error) {
	enc := []byte{}

	length, err := scale2.Marshal(len(voters))
	if err != nil {
		return nil, err
	}
	enc = append(enc, length...)
	for _, val := range voters {
		e, err := val.Encode()
		if err != nil {
			return nil, err
		}
		enc = append(enc, e...)
	}

	return enc, nil
}

// DecodeGrandpaVoters returns a decoded GrandpaVoters
func DecodeGrandpaVoters(r io.Reader) (GrandpaVoters, error) {
	sd := &scale.Decoder{Reader: r}
	length, err := sd.DecodeInteger()
	if err != nil {
		return nil, err
	}

	voters := make([]GrandpaVoter, length)
	for i := range voters {
		voters[i] = GrandpaVoter{}
		err = voters[i].Decode(r)
		if err != nil {
			return nil, err
		}
	}

	return voters, nil
}

// FinalisationInfo represents information about what block was finalised in what round and setID
type FinalisationInfo struct {
	Header Header
	Round  uint64
	SetID  uint64
}

// GrandpaSignedVote represents a signed precommit message for a finalised block
type GrandpaSignedVote struct {
	Vote        GrandpaVote
	Signature   [64]byte
	AuthorityID ed25519.PublicKeyBytes
}

func (s *GrandpaSignedVote) String() string {
	return fmt.Sprintf("SignedVote hash=%s number=%d authority=%s",
		s.Vote.Hash,
		s.Vote.Number,
		s.AuthorityID,
	)
}

// GrandpaVote represents a vote for a block with the given hash and number
type GrandpaVote struct {
	Hash   common.Hash
	Number uint32
}

// String returns the Vote as a string
func (v *GrandpaVote) String() string {
	return fmt.Sprintf("hash=%s number=%d", v.Hash, v.Number)
}
