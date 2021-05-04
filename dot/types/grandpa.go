package types

import (
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/scale"
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
func (a *Authority) FromRawEd25519(raw *GrandpaAuthoritiesRaw) error {
	key, err := ed25519.NewPublicKey(raw.Key[:])
	if err != nil {
		return err
	}

	a.Key = key
	a.Weight = raw.ID
	return nil
}

// GrandpaAuthoritiesRawToAuthorities turns a slice of GrandpaAuthoritiesRaw into a slice of Authority
func GrandpaAuthoritiesRawToAuthorities(adr []*GrandpaAuthoritiesRaw) ([]*Authority, error) {
	ad := make([]*Authority, len(adr))
	for i, r := range adr {
		ad[i] = new(Authority)
		err := ad[i].FromRawEd25519(r)
		if err != nil {
			return nil, err
		}
	}

	return ad, nil
}

// GrandpaVoter represents a GRANDPA voter
type GrandpaVoter struct {
	Key *ed25519.PublicKey
	ID  uint64
}

// PublicKeyBytes returns the voter key as PublicKeyBytes
func (v *GrandpaVoter) PublicKeyBytes() ed25519.PublicKeyBytes {
	return v.Key.AsBytes()
}

// String returns a formatted GrandpaVoter string
func (v *GrandpaVoter) String() string {
	return fmt.Sprintf("[key=0x%s id=%d]", v.PublicKeyBytes(), v.ID)
}

// Decode will decode the Reader into a GrandpaVoter
func (v *GrandpaVoter) Decode(r io.Reader) error {
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

	v.Key = key
	v.ID = id
	return nil
}

// NewGrandpaVotersFromAuthorities returns an array of GrandpaVoters given an array of GrandpaAuthorities
func NewGrandpaVotersFromAuthorities(ad []*Authority) []*GrandpaVoter {
	v := make([]*GrandpaVoter, len(ad))

	for i, d := range ad {
		if pk, ok := d.Key.(*ed25519.PublicKey); ok {
			v[i] = &GrandpaVoter{
				Key: pk,
				ID:  d.Weight,
			}
		}
	}

	return v
}

// NewGrandpaVotersFromAuthoritiesRaw returns an array of GrandpaVoters given an array of GrandpaAuthoritiesRaw
func NewGrandpaVotersFromAuthoritiesRaw(ad []*GrandpaAuthoritiesRaw) ([]*GrandpaVoter, error) {
	v := make([]*GrandpaVoter, len(ad))

	for i, d := range ad {
		key, err := ed25519.NewPublicKey(d.Key[:])
		if err != nil {
			return nil, err
		}

		v[i] = &GrandpaVoter{
			Key: key,
			ID:  d.ID,
		}
	}

	return v, nil
}

// GrandpaVoters represents []*GrandpaVoter
type GrandpaVoters []*GrandpaVoter

// String returns a formatted Voters string
func (v GrandpaVoters) String() string {
	str := ""
	for _, w := range v {
		str = str + w.String() + " "
	}
	return str
}

// DecodeGrandpaVoters returns a SCALE decoded GrandpaVoters
func DecodeGrandpaVoters(r io.Reader) (GrandpaVoters, error) {
	sd := &scale.Decoder{Reader: r}
	length, err := sd.DecodeInteger()
	if err != nil {
		return nil, err
	}

	voters := make([]*GrandpaVoter, length)
	for i := range voters {
		voters[i] = new(GrandpaVoter)
		err = voters[i].Decode(r)
		if err != nil {
			return nil, err
		}
	}

	return voters, nil
}

// FinalisationInfo represents information about what block was finalised in what round and setID
type FinalisationInfo struct {
	Header *Header
	Round  uint64
	SetID  uint64
}
