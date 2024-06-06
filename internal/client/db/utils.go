// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"fmt"
	"log"
	"math"

	"github.com/ChainSafe/gossamer/internal/client/db/columns"
	"github.com/ChainSafe/gossamer/internal/client/db/metakeys"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Meta column. The set of keys in the column is shared by full && light storages.
const columnMeta = columns.Meta

// Database metadata.
type meta[H, N any] struct {
	// Hash of the best known block.
	BestHash H
	// Number of the best known block.
	BestNumber N
	// Hash of the best finalized block.
	FinalizedHash H
	// Number of the best finalized block.
	FinalizedNumber N
	// Hash of the genesis block.
	GenesisHash H
	// Finalized state, if any
	FinalizedState *struct {
		Hash   H
		Number N
	}
	// Block gap, start and end inclusive, if any.
	BlockGap *[2]N
}

// A block lookup key: used for canonical lookup from block number to hash
type numberIndexKey [4]byte

// Convert block number into short lookup key (LE representation) for
// blocks that are in the canonical chain.
//
// In the current database schema, this kind of key is only used for
// lookups into an index, NOT for storing header data or others.
func newNumberIndexKey[N runtime.Number](num N) (numberIndexKey, error) {
	if num > N(math.MaxUint32) {
		return numberIndexKey{}, fmt.Errorf("block number cannot be converted to uint32")
	}
	var n uint32 = uint32(num)

	return numberIndexKey{byte(n >> 24), byte((n >> 16) & 0xff), byte((n >> 8) & 0xff), byte(n & 0xff)}, nil
}

// Convert block id to block lookup key.
// block lookup key is the DB-key header, block and justification are stored under.
// looks up lookup key by hash from DB as necessary.
func blockIDToLookupKey[H runtime.Hash, N runtime.Number](
	db database.Database[hash.H256], keyLookupCol database.ColumnID, id generic.BlockID,
) ([]byte, error) {
	switch id := id.(type) {
	case generic.BlockIDNumber[N]:
		key, err := newNumberIndexKey(id.Inner)
		if err != nil {
			return nil, err
		}
		return db.Get(keyLookupCol, key[:]), nil
	case generic.BlockIDHash[H]:
		return db.Get(keyLookupCol, id.Inner.Bytes()), nil
	default:
		panic("wtf?")
	}
}

// Read database column entry for the given block.
func readDB[H runtime.Hash, N runtime.Number](
	db database.Database[hash.H256], colIndex, col database.ColumnID, id generic.BlockID,
) ([]byte, error) {
	key, err := blockIDToLookupKey[H, N](db, colIndex, id)
	if err != nil {
		return nil, err
	}
	if key != nil {
		return db.Get(col, key), nil
	}
	return nil, nil
}

// Read a header from the database.
func readHeader[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]](
	db database.Database[hash.H256], colIndex, col database.ColumnID, id generic.BlockID,
) (*runtime.Header[N, H], error) {
	headerBytes, err := readDB[H, N](db, colIndex, col, id)
	if err != nil {
		return nil, err
	}
	if headerBytes == nil {
		return nil, nil
	}
	var header Header
	err = scale.Unmarshal(headerBytes, &header)
	if err != nil {
		return nil, err
	}
	ret := runtime.Header[N, H](header)
	return &ret, nil
}

func readMeta[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]](
	db database.Database[hash.H256], colHeader database.ColumnID,
) (meta[H, N], error) {
	genesisHash, err := readGenesisHash[H](db)
	if err != nil {
		return meta[H, N]{}, err
	}
	if genesisHash == nil {
		return meta[H, N]{}, nil
	}

	var loadMetaBlock = func(desc string, key []byte) (hash H, number N, err error) {
		id := db.Get(columnMeta, key)
		if id == nil {
			return
		}
		headerBytes := db.Get(colHeader, id)
		if headerBytes == nil {
			return
		}
		var header = new(Header)
		err = scale.Unmarshal(headerBytes, header)
		if err != nil {
			return
		}
		hash = (*header).Hash()
		log.Printf("DEBUG: Opened blockchain db, fetched %v = %v (%v)\n", desc, hash, (*header).Number())
		return hash, (*header).Number(), nil
	}

	bestHash, bestNumber, err := loadMetaBlock("best", metakeys.BestBlock)
	if err != nil {
		return meta[H, N]{}, err
	}
	finalizedHash, finalizedNumber, err := loadMetaBlock("final", metakeys.FinalizedBlock)
	if err != nil {
		return meta[H, N]{}, err
	}
	finalizedStateHash, finalizedStateNumber, err := loadMetaBlock("final_state", metakeys.FinalizedState)
	if err != nil {
		return meta[H, N]{}, err
	}
	var finalizedState *struct {
		Hash   H
		Number N
	}
	if finalizedStateHash != *new(H) {
		finalizedState = &struct {
			Hash   H
			Number N
		}{
			finalizedStateHash, finalizedStateNumber,
		}
	}
	var blockGap *[2]N
	blockGapBytes := db.Get(columnMeta, metakeys.BlockGap)
	if blockGapBytes != nil {
		err = scale.Unmarshal(blockGapBytes, blockGap)
		if err != nil {
			return meta[H, N]{}, err
		}
	}

	return meta[H, N]{
		BestHash:        bestHash,
		BestNumber:      bestNumber,
		FinalizedHash:   finalizedHash,
		FinalizedNumber: finalizedNumber,
		GenesisHash:     *genesisHash,
		FinalizedState:  finalizedState,
		BlockGap:        blockGap,
	}, nil
}

// Read genesis hash from database.
func readGenesisHash[H any](db database.Database[hash.H256]) (*H, error) {
	val := db.Get(columnMeta, metakeys.GenesisHash)
	if val != nil {
		var h H
		err := scale.Unmarshal(val, &h)
		if err != nil {
			return nil, err
		}
		return &h, nil
	}
	return nil, nil
}

type joinInput struct {
	I1 []byte
	I2 []byte
}

func (ji joinInput) Bytes() []byte {
	return append(ji.I1, ji.I2...)
}

func newJoinInput(i1 []byte, i2 []byte) joinInput {
	return joinInput{i1, i2}
}
