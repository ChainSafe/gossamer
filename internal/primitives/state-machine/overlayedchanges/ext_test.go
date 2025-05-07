package overlayedchanges

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

func TestNextStorageKeyWorks(t *testing.T) {
	overlay := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	overlay.SetStorage([]byte{20}, nil)
	overlay.SetStorage([]byte{30}, []byte{31})

	top := btree.Map[string, []byte]{}
	top.Set(string([]byte{10}), []byte{10})
	top.Set(string([]byte{20}), []byte{20})
	top.Set(string([]byte{40}), []byte{40})
	backend := statemachine.NewMemoryDBTrieBackendFromStorage[hash.H256, runtime.BlakeTwo256](storage.Storage{
		Top: top,
	}, storage.StateVersionV1)

	ext := NewExt(*overlay, backend)

	// next_backend < next_overlay
	require.Equal(t, []byte{10}, ext.NextStorageKey([]byte{5}))
	// next_backend == next_overlay but next_overlay is a delete
	require.Equal(t, []byte{30}, ext.NextStorageKey([]byte{10}))
	// next_overlay < next_backend
	require.Equal(t, []byte{30}, ext.NextStorageKey([]byte{20}))
	// next_backend exist but next_overlay doesn't exist
	require.Equal(t, []byte{40}, ext.NextStorageKey([]byte{30}))

	overlay.SetStorage([]byte{50}, []byte{50})
	ext = NewExt(*overlay, backend)

	// next_overlay exist but next_backend doesn't exist
	require.Equal(t, []byte{50}, ext.NextStorageKey([]byte{40}))
}

func TestNextStorageKeyWorksWithALotEmptyValuesInOverlay(t *testing.T) {
	overlay := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()
	overlay.SetStorage([]byte{20}, nil)
	overlay.SetStorage([]byte{21}, nil)
	overlay.SetStorage([]byte{22}, nil)
	overlay.SetStorage([]byte{23}, nil)
	overlay.SetStorage([]byte{24}, nil)
	overlay.SetStorage([]byte{25}, nil)
	overlay.SetStorage([]byte{26}, nil)
	overlay.SetStorage([]byte{27}, nil)
	overlay.SetStorage([]byte{28}, nil)
	overlay.SetStorage([]byte{29}, nil)

	top := btree.Map[string, []byte]{}
	top.Set(string([]byte{30}), []byte{30})

	backend := statemachine.NewMemoryDBTrieBackendFromStorage[hash.H256, runtime.BlakeTwo256](storage.Storage{
		Top: top,
	}, storage.StateVersionV1)

	ext := NewExt(*overlay, backend)

	require.Equal(t, []byte{30}, ext.NextStorageKey([]byte{5}))
}

func TestNextChildStorageKeyWorks(t *testing.T) {
	childInfo := storage.NewDefaultChildInfo([]byte("Child1"))

	overlay := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	overlay.SetChildStorage(childInfo, []byte{20}, nil)
	overlay.SetChildStorage(childInfo, []byte{30}, []byte{31})

	childData := btree.Map[string, []byte]{}
	childData.Set(string([]byte{10}), []byte{10})
	childData.Set(string([]byte{20}), []byte{20})
	childData.Set(string([]byte{40}), []byte{40})
	backend := statemachine.NewMemoryDBTrieBackendFromStorage[hash.H256, runtime.BlakeTwo256](storage.Storage{
		ChildrenDefault: map[string]storage.StorageChild{
			string(childInfo.Keyspace()): {
				Data:      childData,
				ChildInfo: childInfo,
			},
		},
	}, storage.StateVersionV1)

	ext := NewExt(*overlay, backend)

	// next_backend < next_overlay
	require.Equal(t, []byte{10}, ext.NextChildStorageKey(childInfo, []byte{5}))
	// next_backend == next_overlay but next_overlay is a delete
	require.Equal(t, []byte{30}, ext.NextChildStorageKey(childInfo, []byte{10}))
	// next_overlay < next_backend
	require.Equal(t, []byte{30}, ext.NextChildStorageKey(childInfo, []byte{20}))
	// next_backend exist but next_overlay doesn't exist
	require.Equal(t, []byte{40}, ext.NextChildStorageKey(childInfo, []byte{30}))

	overlay.SetChildStorage(childInfo, []byte{50}, []byte{50})
	ext = NewExt(*overlay, backend)

	// next_overlay exist but next_backend doesn't exist
	require.Equal(t, []byte{50}, ext.NextChildStorageKey(childInfo, []byte{40}))
}

func TestChildStorageWorks(t *testing.T) {
	childInfo := storage.NewDefaultChildInfo([]byte("Child1"))
	overlay := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	overlay.SetChildStorage(childInfo, []byte{20}, nil)
	overlay.SetChildStorage(childInfo, []byte{30}, []byte{31})

	childData := btree.Map[string, []byte]{}
	childData.Set(string([]byte{10}), []byte{10})
	childData.Set(string([]byte{20}), []byte{20})
	childData.Set(string([]byte{30}), []byte{40})
	backend := statemachine.NewMemoryDBTrieBackendFromStorage[hash.H256, runtime.BlakeTwo256](storage.Storage{
		ChildrenDefault: map[string]storage.StorageChild{
			string(childInfo.Keyspace()): {
				Data:      childData,
				ChildInfo: childInfo,
			},
		},
	}, storage.StateVersionV1)

	ext := NewExt(*overlay, backend)

	require.Equal(t, []byte{10}, ext.ChildStorage(childInfo, []byte{10}))
	require.Equal(t, runtime.BlakeTwo256{}.Hash([]byte{10}).Bytes(), ext.ChildStorageHash(childInfo, []byte{10}))

	require.Equal(t, []byte(nil), ext.ChildStorage(childInfo, []byte{20}))
	require.Equal(t, []byte(nil), ext.ChildStorageHash(childInfo, []byte{20}))

	require.Equal(t, []byte{31}, ext.ChildStorage(childInfo, []byte{30}))
	require.Equal(t, runtime.BlakeTwo256{}.Hash([]byte{31}).Bytes(), ext.ChildStorageHash(childInfo, []byte{30}))
}

func TestClearPrefixCannotDeleteAChildRoot(t *testing.T) {
	childInfo := storage.NewDefaultChildInfo([]byte("Child1"))
	overlay := NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

	childData := btree.Map[string, []byte]{}
	childData.Set(string([]byte{30}), []byte{40})
	backend := statemachine.NewMemoryDBTrieBackendFromStorage[hash.H256, runtime.BlakeTwo256](storage.Storage{
		ChildrenDefault: map[string]storage.StorageChild{
			string(childInfo.Keyspace()): {
				Data:      childData,
				ChildInfo: childInfo,
			},
		},
	}, storage.StateVersionV1)

	ext := NewExt(*overlay, backend)

	notUnderPrefix := keys.ChildStorageKeyPrefix
	notUnderPrefix[4] = 88
	notUnderPrefix = append(notUnderPrefix, []byte("path")...)
	ext.SetStorage(notUnderPrefix, []byte{10})
}
