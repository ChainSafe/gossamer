package proof

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

// storageValueDatabase writes and reads node storage values in-memory.
type storageValueDatabase struct {
	hashToBytes map[common.Hash][]byte
}

func newStorageValueDatabase() *storageValueDatabase {
	return &storageValueDatabase{
		hashToBytes: make(map[common.Hash][]byte),
	}
}

var (
	ErrStorageValueNotFound = errors.New("storage value not found")
)

func (svdb *storageValueDatabase) Get(storageValueHash []byte) (
	storageValue []byte, err error) {
	storageValue, ok := svdb.hashToBytes[common.NewHash(storageValueHash)]
	if !ok {
		return nil, fmt.Errorf("%w: at key 0x%x", ErrStorageValueNotFound, storageValueHash)
	}

	return storageValue, nil
}

func (svdb *storageValueDatabase) Put(storageValueHash, storageValue []byte) (err error) {
	mapKey := common.NewHash(storageValueHash)
	svdb.hashToBytes[mapKey] = storageValue
	return nil
}

// hybridDatabase retrieves nodes from the persistent
// database injected to it and writes and retrieves
// storage values to and from an in-memory mapping.
type hybridDatabase struct {
	persistentDB   Database
	storageValueDB map[common.Hash][]byte
}

func newHybridDatabase(persistentDB Database) *hybridDatabase {
	return &hybridDatabase{
		persistentDB:   persistentDB,
		storageValueDB: make(map[common.Hash][]byte),
	}
}

// Put is for putting subvalues ONLY.
func (h *hybridDatabase) Put(storageValueHash, storageValue []byte) (err error) {
	h.storageValueDB[common.NewHash(storageValueHash)] = storageValue
	return nil
}

// Get tries to retrieve a storage value from the in-memory database map
// first and then tries to retrieve an encoded node from the persistent
// database.
func (h *hybridDatabase) Get(key []byte) (storageValue []byte, err error) {
	storageValue, ok := h.storageValueDB[common.NewHash(key)]
	if ok {
		return storageValue, nil
	}
	return h.persistentDB.Get(key)
}
