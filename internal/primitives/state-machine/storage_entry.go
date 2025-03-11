package statemachine

// Content in an overlay for a given transactional depth.
type StorageEntry interface {
	isStorageEntry()
}

type (
	// The storage entry should be set to the stored value.
	SetStorageEntry struct {
		StorageValue
	}

	// The storage entry should be removed.
	RemoveStorageEntry struct{}

	// The storage entry was appended to.
	//
	// This assumes that the storage entry is encoded as a SCALE list. This means that it is
	// prefixed with a `Compact<u32>` that reprensents the length, followed by all the encoded
	// elements.
	AppendStorageEntry struct {
		// The value of the storage entry.
		// This may or may not be prefixed by the length, depending on the materialized length.
		data StorageValue
		// Current number of elements stored in data.
		currentLength uint32
		// The number of elements as stored in the prefixed length in `data`.
		// If `nil`, than `data` is not yet prefixed with the length.
		materializedLength *uint32
		// The size of `data` in the parent transactional layer.
		// Only set when the parent layer is in  `Append` state.
		parentSize *uint
	}
)

func (SetStorageEntry) isStorageEntry()    {}
func (RemoveStorageEntry) isStorageEntry() {}
func (AppendStorageEntry) isStorageEntry() {}
