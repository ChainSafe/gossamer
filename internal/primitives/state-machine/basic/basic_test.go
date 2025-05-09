// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package basic

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

func TestCommitShouldWork(t *testing.T) {
	ext := NewEmptyBasicExternalities()
	ext.SetStorage([]byte("doe"), []byte("reindeer"))
	ext.SetStorage([]byte("dog"), []byte("puppy"))
	ext.SetStorage([]byte("dogglesworth"), []byte("cat"))

	expectedRoot := common.MustHexToBytes("0x39245109cef3758c2eed2ccba8d9b370a917850af3824bc8348d505df2c298fa")
	root := ext.StorageRoot(storage.StateVersionV1)

	require.Equal(t, expectedRoot, root)
}

func TestSetAndRetrieveCode(t *testing.T) {
	ext := NewEmptyBasicExternalities()

	code := []byte{1, 2, 3}
	ext.SetStorage(keys.Code, code)

	codeFromStorage := ext.Storage(keys.Code)

	require.NotNil(t, codeFromStorage)
	require.Equal(t, code, codeFromStorage)
}

func TestChildrenWorks(t *testing.T) {
	childInfo := storage.NewDefaultChildInfo([]byte("storage_key"))

	childStorageData := btree.Map[string, []byte]{}
	childStorageData.Set("doe", []byte("reindeer"))

	ext := NewBasicExternalities(storage.Storage{
		Top: btree.Map[string, []byte]{},
		ChildrenDefault: map[string]storage.StorageChild{
			string(childInfo.StorageKey()): {
				Data:      childStorageData,
				ChildInfo: childInfo,
			},
		},
	})

	require.Equal(t, ext.ChildStorage(childInfo, []byte("doe")), []byte("reindeer"))

	ext.SetChildStorage(childInfo, []byte("dog"), []byte("puppy"))
	require.Equal(t, ext.ChildStorage(childInfo, []byte("dog")), []byte("puppy"))

	ext.ClearChildStorage(childInfo, []byte("dog"))
	require.Nil(t, ext.ChildStorage(childInfo, []byte("dog")))

	ext.KillChildStorage(childInfo, nil, nil)
	require.Nil(t, ext.ChildStorage(childInfo, []byte("doe")))
}

func TestKillChildStorageReturnsNumElementsRemoved(t *testing.T) {
	childInfo := storage.NewDefaultChildInfo([]byte("storage_key"))

	childStorageData := btree.Map[string, []byte]{}
	childStorageData.Set("doe", []byte("reindeer"))
	childStorageData.Set("dog", []byte("puppy"))
	childStorageData.Set("hello", []byte("world"))

	ext := NewBasicExternalities(storage.Storage{
		Top: btree.Map[string, []byte]{},
		ChildrenDefault: map[string]storage.StorageChild{
			string(childInfo.StorageKey()): {
				Data:      childStorageData,
				ChildInfo: childInfo,
			},
		},
	})

	res := ext.KillChildStorage(childInfo, nil, nil)
	require.Nil(t, res.Cursor)
	require.Equal(t, uint32(3), res.Backend)
	require.Equal(t, uint32(3), res.Unique)
	require.Equal(t, uint32(3), res.Loops)
}

func TestBasicExternalitiesIsEmpty(t *testing.T) {
	storage := NewEmptyBasicExternalities().IntoStorages()

	require.Empty(t, storage.Top)
	require.Empty(t, storage.ChildrenDefault)
}
