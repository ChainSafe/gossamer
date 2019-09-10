package common

import (
	"golang.org/x/crypto/blake2b"
)

func Blake2b128(in []byte) ([]byte, error) {
	hasher, err := blake2b.New(16, nil) 
	if err != nil {
		return nil, err
	}

	return hasher.Sum(in)[:16], nil
}

func Blake2bHash(in []byte) (Hash, error) {
	h, err := blake2b.New256(nil)
	if err != nil {
		return [32]byte{}, err
	}

	var res []byte
	_, err = h.Write(in)
	if err != nil {
		return [32]byte{}, err
	}

	res = h.Sum(nil)
	var buf = [32]byte{}
	copy(buf[:], res)
	return buf, err
}
