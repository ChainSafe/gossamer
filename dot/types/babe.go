package types

import (
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
)

// BabeConfiguration contains the genesis data for BABE
// see: https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/consensus/babe/primitives/src/lib.rs#L132
type BabeConfiguration struct {
	SlotDuration       uint64 // milliseconds
	EpochLength        uint64 // duration of epoch in slots
	C1                 uint64 // (1-(c1/c2)) is the probability of a slot being empty
	C2                 uint64
	// todo ed authorities
	GenesisAuthorities []*AuthorityRaw
	Randomness         [32]byte
	SecondarySlots     bool
}

// BABEAuthorityDataRaw represents a BABE authority where their key is a byte array
//type BABEAuthorityDataRaw struct {
//	ID     [sr25519.PublicKeyLength]byte
//	Weight uint64
//}

// Decode will decode the Reader into a BABEAuthorityDataRaw
// todo ed authorities
func (a *AuthorityRaw) Decode(r io.Reader) (*AuthorityRaw, error) {
	id, err := common.Read32Bytes(r)
	if err != nil {
		return nil, err
	}

	weight, err := common.ReadUint64(r)
	if err != nil {
		return nil, err
	}

	a = new(AuthorityRaw)
	a.Key = id
	a.Weight = weight

	return a, nil
}

//// BABEAuthorityData represents a BABE authority
//type BABEAuthorityData struct {
//	ID     *sr25519.PublicKey
//	Weight uint64
//}

////NewBABEAuthorityData returns BABEAuthorityData with the given id and weight
//func NewBABEAuthorityData(pub *sr25519.PublicKey, weight uint64) *BABEAuthorityData {
//	return &BABEAuthorityData{
//		ID:     pub,
//		Weight: weight,
//	}
//}

//// ToRaw returns the BABEAuthorityData as BABEAuthorityDataRaw. It encodes the authority public keys.
//func (a *BABEAuthorityData) ToRaw() *BABEAuthorityDataRaw {
//	raw := new(BABEAuthorityDataRaw)
//
//	id := a.ID.Encode()
//	copy(raw.ID[:], id)
//
//	raw.Weight = a.Weight
//	return raw
//}

//// Encode returns the SCALE encoding of the BABEAuthorityData.
//func (a *BABEAuthorityData) Encode() []byte {
//	raw := a.ToRaw()
//
//	enc := raw.ID[:]
//
//	weightBytes := make([]byte, 8)
//	binary.LittleEndian.PutUint64(weightBytes, raw.Weight)
//
//	return append(enc, weightBytes...)
//}

//// Decode sets the BABEAuthorityData to the SCALE decoded input.
//func (a *BABEAuthorityData) Decode(r io.Reader) error {
//	id, err := common.Read32Bytes(r)
//	if err != nil {
//		return err
//	}
//
//	weight, err := common.ReadUint64(r)
//	if err != nil {
//		return err
//	}
//
//	raw := &BABEAuthorityDataRaw{
//		ID:     id,
//		Weight: weight,
//	}
//
//	return a.FromRaw(raw)
//}

// BABEAuthorityDataRawToAuthorityData turns a slice of BABEAuthorityDataRaw into a slice of BABEAuthorityData
// todo ed authorities
func BABEAuthorityDataRawToAuthorityData(adr []*AuthorityRaw) ([]*Authority, error) {
	// todo ed authorities
	ad := make([]*Authority, len(adr))
	for i, r := range adr {
		// todo ed authorities
		ad[i] = new(Authority)
		err := ad[i].FromRawSr25519(r)
		if err != nil {
			return nil, err
		}
	}

	return ad, nil
}

// RandomnessLength is the length of the epoch randomness (32 bytes)
const RandomnessLength = 32

// EpochInfo is internal BABE information for a given epoch
type EpochInfo struct {
	Duration   uint64 // number of slots in the epoch
	FirstBlock uint64 // number of the first block in the epoch
	Randomness [RandomnessLength]byte
}
