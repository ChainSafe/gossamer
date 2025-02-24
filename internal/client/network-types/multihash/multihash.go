package multihash

import "github.com/multiformats/go-multihash"

// Code is a multihash code
type Code uint64

const (
	Identity  Code = multihash.IDENTITY
	ShaTwo256 Code = multihash.SHA2_256
)

// Multihash is the default multihash implementation. Only hashes used by substrate are defined.
type Multihash struct {
	multihash.Multihash
}

func (m Multihash) Code() uint64 {
	decoded, err := multihash.Decode([]byte(m.Multihash))
	if err != nil {
		panic("unable to decode")
	}
	return decoded.Code
}

func (m Multihash) Digest() []byte {
	decoded, err := multihash.Decode([]byte(m.Multihash))
	if err != nil {
		panic("unable to decode")
	}
	return decoded.Digest
}

func Wrap(code uint64, inputDigest []byte) (Multihash, error) {
	mh, err := multihash.Encode(inputDigest, code)
	if err != nil {
		return Multihash{}, err
	}
	return Multihash{
		Multihash: multihash.Multihash(mh),
	}, nil
}

func NewMultihashFromBytes(b []byte) (Multihash, error) {
	_, err := multihash.Decode(b)
	if err != nil {
		return Multihash{}, err
	}
	return Multihash{
		Multihash: multihash.Multihash(b),
	}, nil
}
