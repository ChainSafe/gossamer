package rawdb

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/polkadb"
)

func setup() (polkadb.Database, *types.BlockHeader) {
	h := &types.BlockHeader{
		ParentHash:     common.BytesToHash([]byte("parent_test")),
		Number:         big.NewInt(2),
		StateRoot:      common.BytesToHash([]byte("state_root_test")),
		ExtrinsicsRoot: common.BytesToHash([]byte("extrinsics_test")),
		Digest:         []byte("digest_test"),
		Hash:           common.BytesToHash([]byte("hash_test")),
	}
	return polkadb.NewMemDatabase(), h
}

func TestSetHeader(t *testing.T) {
	memDB, h := setup()

	SetHeader(memDB, h)
	entry := GetHeader(memDB, h.Hash)
	if entry.Hash != h.Hash {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, h)
	}
}

func TestSetBlockData(t *testing.T) {
	var body *types.BlockBody
	memDB, h := setup()

	bd := &types.BlockData{
		Hash:   common.BytesToHash([]byte("bd_hash")),
		Header: h,
		Body:   body,
	}

	SetBlockData(memDB, bd)
	entry := GetBlockData(memDB, bd.Hash)
	if entry.Hash != bd.Hash {
		t.Fatalf("Retrieved blockData mismatch: have %v, want %v", entry, bd)
	}
}
