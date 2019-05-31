package common

import (
	"math/big"
	"testing"
)

func TestHashBlockHeader(t *testing.T) {
	header := &BlockHeader{
		ParentHash: [32]byte{},
		Number: big.NewInt(1),
		StateRoot: [32]byte{},
		ExtrinsicsRoot: [32]byte{},
		Digest: []byte{},
	}

	hash, err := header.Hash()
	if err != nil {
		t.Errorf("Fail: could not hash block header: %s", err)
	}
	t.Log(hash)
}