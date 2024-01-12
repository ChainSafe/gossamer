// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// GrandpaAuthoritiesRaw represents a GRANDPA authority where their key is a byte array
type GrandpaAuthoritiesRaw struct {
	Key [ed25519.PublicKeyLength]byte
	ID  uint64
}

func (g GrandpaAuthoritiesRaw) String() string {
	return fmt.Sprintf("GrandpaAuthoritiesRaw{Key=0x%x, ID=%d}", g.Key, g.ID)
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

// voter represents a scale compatible GRANDPA voter
type voter struct {
	Key [32]byte
	ID  uint64
}

// GrandpaVoter converts a voter to a GrandpaVoter
func (sv *voter) GrandpaVoter() (GrandpaVoter, error) {
	key, err := ed25519.NewPublicKey(sv.Key[:])
	if err != nil {
		return GrandpaVoter{}, err
	}
	voter := GrandpaVoter{
		Key: *key,
		ID:  sv.ID,
	}
	return voter, nil
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
	return fmt.Sprintf("[key=%s id=%d]", gv.PublicKeyBytes(), gv.ID)
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

// GrandpaVoters represents []GrandpaVoter
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
	sv := make([]voter, len(voters))
	for i := range voters {
		sv[i] = voter{
			Key: voters[i].Key.AsBytes(),
			ID:  voters[i].ID,
		}
	}

	enc, err := scale.Marshal(sv)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

// DecodeGrandpaVoters returns a decoded GrandpaVoters
func DecodeGrandpaVoters(in []byte) (GrandpaVoters, error) {
	var dec []voter
	err := scale.Unmarshal(in, &dec)
	if err != nil {
		return nil, err
	}

	gv := make(GrandpaVoters, len(dec))
	for i := range dec {
		gv[i], err = dec[i].GrandpaVoter()
		if err != nil {
			return nil, err
		}
	}
	return gv, nil
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

func (v GrandpaVote) String() string {
	return fmt.Sprintf("hash=%s number=%d", v.Hash, v.Number)
}

// GrandpaEquivocation is used to create a proof of equivocation
// https://github.com/paritytech/finality-grandpa/blob/19d251d0b0105d51a79d3c4532a9aae75a5035bd/src/lib.rs#L213 //nolint:lll
type GrandpaEquivocation struct {
	RoundNumber     uint64
	ID              [32]byte
	FirstVote       GrandpaVote
	FirstSignature  [64]byte
	SecondVote      GrandpaVote
	SecondSignature [64]byte
}

// GrandpaEquivocationEnum is a wrapper object for GRANDPA equivocation proofs, useful for unifying prevote
// and precommit equivocations under a common type.
// https://github.com/paritytech/substrate/blob/fb22096d2ec6bf38e67ce811ad2c31415237a9a5/primitives/finality-grandpa/src/lib.rs#L272 //nolint:lll
type GrandpaEquivocationEnum struct {
	inner any
}
type GrandpaEquivocationEnumValues interface {
	PreVote | PreCommit
}

func setGrandpaEquivocationEnum[Value GrandpaEquivocationEnumValues](mvdt *GrandpaEquivocationEnum, value Value) {
	mvdt.inner = value
}

func (mvdt *GrandpaEquivocationEnum) SetValue(value any) (err error) {
	switch value := value.(type) {
	case PreVote:
		setGrandpaEquivocationEnum(mvdt, value)
		return
	case PreCommit:
		setGrandpaEquivocationEnum(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt GrandpaEquivocationEnum) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case PreVote:
		return 0, mvdt.inner, nil
	case PreCommit:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt GrandpaEquivocationEnum) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt GrandpaEquivocationEnum) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return PreVote{}, nil
	case 1:
		return PreCommit{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// // Set sets a VaryingDataTypeValue using the underlying VaryingDataType
// func (ge *GrandpaEquivocationEnum) Set(value scale.VaryingDataTypeValue) (err error) {
// 	vdt := scale.VaryingDataType(*ge)
// 	err = vdt.Set(value)
// 	if err != nil {
// 		return err
// 	}
// 	*ge = GrandpaEquivocationEnum(vdt)
// 	return nil
// }

// // Value will return the value from the underlying VaryingDataType
// func (ge *GrandpaEquivocationEnum) Value() (value scale.VaryingDataTypeValue, err error) {
// 	vdt := scale.VaryingDataType(*ge)
// 	return vdt.Value()
// }

// NewGrandpaEquivocation returns a new VaryingDataType to represent a grandpa Equivocation
func NewGrandpaEquivocation() *GrandpaEquivocationEnum {
	return &GrandpaEquivocationEnum{}
}

// PreVote equivocation type for a prevote
type PreVote GrandpaEquivocation

// PreCommit equivocation type for a precommit
type PreCommit GrandpaEquivocation

// GrandpaOpaqueKeyOwnershipProof contains a key ownership proof for reporting equivocations
// https://github.com/paritytech/substrate/blob/fb22096d2ec6bf38e67ce811ad2c31415237a9a5/primitives/finality-grandpa/src/lib.rs#L533 //nolint:lll
type GrandpaOpaqueKeyOwnershipProof []byte

// GrandpaEquivocationProof is used to report grandpa equivocations
// https://github.com/paritytech/substrate/blob/fb22096d2ec6bf38e67ce811ad2c31415237a9a5/primitives/finality-grandpa/src/lib.rs#L238 //nolint:lll
type GrandpaEquivocationProof struct {
	SetID        uint64
	Equivocation GrandpaEquivocationEnum
}
