package hash

import (
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
	*h256 = H256(arr[:])
	return nil
}
