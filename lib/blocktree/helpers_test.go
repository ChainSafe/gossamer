// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

var zeroHash = common.MustHexToHash("0x00")
var testHeader = &types.Header{
	ParentHash: zeroHash,
	Number:     0,
	Digest:     types.NewDigest(),
}

type testBranch struct {
	hash        Hash
	number      uint
	arrivalTime int64
}

func createPrimaryBABEDigest(t testing.TB) types.Digest {
	babeDigest := types.NewBabeDigest()
	err := babeDigest.SetValue(types.BabePrimaryPreDigest{AuthorityIndex: 0})
	require.NoError(t, err)

	bdEnc, err := scale.Marshal(babeDigest)
	require.NoError(t, err)

	digest := types.NewDigest()
	err = digest.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              bdEnc,
	})
	require.NoError(t, err)
	return digest
}

func createTestBlockTree(t *testing.T, header *types.Header, number uint) (*BlockTree, []testBranch) {
	bt := NewBlockTreeFromRoot(header)
	previousHash := header.Hash()

	// branch tree randomly
	var branches []testBranch
	r := rand.New(rand.NewSource(time.Now().UnixNano())) // skipcq

	at := int64(0)

	// create base tree
	for i := uint(1); i <= number; i++ {
		header := &types.Header{
			ParentHash: previousHash,
			Number:     i,
			Digest:     createPrimaryBABEDigest(t),
		}

		hash := header.Hash()
		err := bt.AddBlock(header, time.Unix(0, at))
		require.NoError(t, err)

		previousHash = hash

		isBranch := r.Intn(2)
		if isBranch == 1 {
			branches = append(branches, testBranch{
				hash:        hash,
				number:      bt.getNode(hash).number,
				arrivalTime: at,
			})
		}

		at += int64(r.Intn(8))
	}

	// create tree branches
	for _, branch := range branches {
		at := branch.arrivalTime
		previousHash = branch.hash

		for i := branch.number; i <= number; i++ {
			header := &types.Header{
				ParentHash: previousHash,
				Number:     i + 1,
				StateRoot:  common.Hash{0x1},
				Digest:     createPrimaryBABEDigest(t),
			}

			hash := header.Hash()
			err := bt.AddBlock(header, time.Unix(0, at))
			require.NoError(t, err)

			previousHash = hash
			at += int64(r.Intn(8))

		}
	}

	return bt, branches
}

func createFlatTree(t testing.TB, number uint) (*BlockTree, []common.Hash) {
	rootHeader := &types.Header{
		ParentHash: zeroHash,
		Digest:     createPrimaryBABEDigest(t),
	}

	bt := NewBlockTreeFromRoot(rootHeader)
	require.NotNil(t, bt)
	previousHash := bt.root.hash

	hashes := []common.Hash{bt.root.hash}
	for i := uint(1); i <= number; i++ {
		header := &types.Header{
			ParentHash: previousHash,
			Number:     i,
			Digest:     createPrimaryBABEDigest(t),
		}

		hash := header.Hash()
		hashes = append(hashes, hash)

		err := bt.AddBlock(header, time.Unix(0, 0))
		require.NoError(t, err)
		previousHash = hash
	}

	return bt, hashes
}
