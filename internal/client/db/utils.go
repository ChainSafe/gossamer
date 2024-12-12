// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"

	"github.com/ChainSafe/gossamer/internal/client/db/columns"
	"github.com/ChainSafe/gossamer/internal/client/db/metakeys"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// / Number of columns in the db. Must be the same for both full && light dbs.
// / Otherwise RocksDb will fail to open database && check its type.
const NumColumns uint32 = 13

// Meta column. The set of keys in the column is shared by full && light storages.
const columnMeta = columns.Meta

type finalizedState[H, N any] struct {
	Hash   H
	Number N
}

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
	FinalizedState *finalizedState[H, N]
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

// / Convert number and hash into long lookup key for blocks that are
// / not in the canonical chain.
func newLookupKey[N runtime.Number, H runtime.Hash](number N, hash H) ([]byte, error) {
	lookupKey, err := newNumberIndexKey(number)
	if err != nil {
		return nil, err
	}
	key := append(lookupKey[:], hash.Bytes()...)
	return key, nil
}

// / Delete number to hash mapping in DB transaction.
func removeNumberToKeyMapping[N runtime.Number](
	transaction *database.Transaction[hash.H256], keyLookupCol uint32, number N,
) error {
	lookupKey, err := newNumberIndexKey(number)
	if err != nil {
		return err
	}
	transaction.Remove(database.ColumnID(keyLookupCol), lookupKey[:])
	return nil
}

// / Place a number mapping into the database. This maps number to current perceived
// / block hash at that position.
func insertNumberToKeyMapping[H runtime.Hash, N runtime.Number](
	transaction *database.Transaction[hash.H256], keyLookupCol uint32, number N, hash H,
) error {
	numberIndexKey, err := newNumberIndexKey(number)
	if err != nil {
		return err
	}
	lookupKey, err := newLookupKey(number, hash)
	if err != nil {
		return err
	}
	transaction.Set(database.ColumnID(keyLookupCol), numberIndexKey[:], lookupKey)
	return nil
}

// / Insert a hash to key mapping in the database.
func insertHashToKeyMapping[H runtime.Hash, N runtime.Number](
	transaction *database.Transaction[hash.H256], keyLookupCol uint32, number N, hash H,
) error {
	lookupKey, err := newLookupKey(number, hash)
	if err != nil {
		return err
	}
	transaction.Set(database.ColumnID(keyLookupCol), hash.Bytes(), lookupKey)
	return nil
}

// Convert block id to block lookup key.
// block lookup key is the DB-key header, block and justification are stored under.
// looks up lookup key by hash from DB as necessary.
func blockIDToLookupKey[H runtime.Hash, N runtime.Number](
	db database.Database[hash.H256], keyLookupCol database.ColumnID, id generic.BlockID,
) ([]byte, error) {
	switch id := id.(type) {
	case generic.BlockIDNumber[N]:
		key, err := newNumberIndexKey(id.Number)
		if err != nil {
			return nil, err
		}
		return db.Get(keyLookupCol, key[:]), nil
	case generic.BlockIDHash[H]:
		return db.Get(keyLookupCol, id.Hash.Bytes()), nil
	default:
		panic("unsupported generic.BlockID")
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

// / Remove database column entry for the given block.
func removeFromDB[H runtime.Hash, N runtime.Number](
	transaction *database.Transaction[hash.H256],
	db database.Database[hash.H256],
	colIndex uint32,
	col uint32,
	id generic.BlockID,
) error {
	key, err := blockIDToLookupKey[H, N](db, database.ColumnID(colIndex), id)
	if err != nil {
		return err
	}
	if key != nil {
		transaction.Remove(database.ColumnID(col), key)
	}
	return nil
}

// Read a header from the database.
func readHeader[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]](
	db database.Database[hash.H256], colIndex, col database.ColumnID, id generic.BlockID,
) (*Header, error) {
	headerBytes, err := readDB[H, N](db, colIndex, col, id)
	if err != nil {
		return nil, err
	}
	if headerBytes == nil {
		return nil, nil
	}
	t := reflect.TypeOf((*new(Header))).Elem()
	header := reflect.New(t).Interface()
	err = scale.Unmarshal(headerBytes, header)
	if err != nil {
		return nil, err
	}
	h := header.(Header)
	return &h, nil
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
		t := reflect.TypeOf((*new(Header))).Elem()
		header := reflect.New(t).Interface()
		err = scale.Unmarshal(headerBytes, header)
		if err != nil {
			return
		}
		h := header.(Header)
		hash = h.Hash()
		log.Printf("DEBUG: Opened blockchain db, fetched %v = %v (%v)", desc, hash, h.Number())
		return hash, h.Number(), nil
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
	var finalized *finalizedState[H, N]
	if finalizedStateHash != *new(H) {
		finalized = &finalizedState[H, N]{
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
		FinalizedState:  finalized,
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

func joinInput(i1 []byte, i2 []byte) []byte {
	return bytes.Join([][]byte{i1, i2}, nil)
}

var (
	errDoesNotExist = errors.New("Database does not exist at given location")
)

func openDatabase(dbSource DatabaseSource, create bool) (database.Database[hash.H256], error) {
	// Maybe migrate (copy) the database to a type specific subdirectory to make it
	// possible that light and full databases coexist
	// NOTE: This function can be removed in a few releases
	// maybe_migrate_to_type_subdir::<Block>(db_source, db_type)?;
	if dbSource.RequireCreateFlag && !create {
		return nil, errDoesNotExist
	}

	return dbSource.DB, nil
}
