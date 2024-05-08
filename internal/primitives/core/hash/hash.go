// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hash

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Fixed-size uninterpreted hash type with 32 bytes (256 bits) size.
type H256 string

func (h256 H256) Bytes() []byte {
	return []byte(h256)
}

func (h256 H256) String() string {
	return fmt.Sprintf("%v", h256.Bytes())
}

func (h256 H256) MarshalSCALE() ([]byte, error) {
	var arr [32]byte
	copy(arr[:], []byte(h256))
	return scale.Marshal(arr)
}

func (h256 *H256) UnmarshalSCALE(r io.Reader) error {
	var arr [32]byte
	decoder := scale.NewDecoder(r)
	err := decoder.Decode(&arr)
	if err != nil {
		return err
	}

	if arr != [32]byte{} {
		*h256 = H256(arr[:])
	}
	return nil
}

func NewH256FromLowUint64BigEndian(v uint64) H256 {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	full := append(b, make([]byte, 24)...)
	return H256(full)
}

func NewRandomH256() H256 {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		panic(err)
	}
	return H256(token)
}
