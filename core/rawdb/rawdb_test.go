package rawdb

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
)

func setup(t *testing.T) (polkadb.Database, *types.BlockHeader) {
	h := &types.BlockHeader{
		ParentHash:     common.BytesToHash([]byte("parent_test")),
		Number:         big.NewInt(2),
		StateRoot:      common.BytesToHash([]byte("state_root_test")),
		ExtrinsicsRoot: common.BytesToHash([]byte("extrinsics_test")),
		Digest:         []byte("digest_test"),
	}
	_, err := h.Hash()
	if err != nil {
		t.Fatal(err)
	}
	return polkadb.NewMemDatabase(), h
}

func TestSetHeader(t *testing.T) {
	memDB, h := setup(t)

	SetHeader(memDB, h)
	entry := GetHeader(memDB, h.GetHash())
	if reflect.DeepEqual(entry, h) {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, h)
	}
	entryHash, err := entry.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if h.GetHash() != entryHash {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, h)
	}
}

func TestSetBlockData(t *testing.T) {
	var body *types.BlockBody
	memDB, h := setup(t)

	hash, err := h.Hash()
	if err != nil {
		t.Fatal(err)
	}

	bd := &types.BlockData{
		Hash:   hash,
		Header: h,
		Body:   body,
	}

	SetBlockData(memDB, bd)
	entry := GetBlockData(memDB, bd.Header.GetHash())
	if reflect.DeepEqual(entry, bd) {
		t.Fatalf("Retrieved blockData mismatch: have %v, want %v", entry, bd)
	}
	if bd.Hash != entry.Hash {
		t.Fatalf("Retrieved blockData mismatch: have %v, want %v", entry, bd)
	}
}
