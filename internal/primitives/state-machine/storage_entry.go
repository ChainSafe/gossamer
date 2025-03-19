package statemachine

// Content in an overlay for a given transactional depth.
type StorageEntry interface {
	value() StorageValue
	optionalValue() StorageValue
}

type (
	// The storage entry should be set to the stored value.
	SetStorageEntry struct {
		data StorageValue
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
		// This may or may not be prefixed by the length, depending on the materialised length.
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

func (se SetStorageEntry) value() StorageValue            { return se.data }
func (se SetStorageEntry) optionalValue() StorageValue    { return se.data }
func (se RemoveStorageEntry) value() StorageValue         { return nil }
func (se RemoveStorageEntry) optionalValue() StorageValue { return nil }
func (se AppendStorageEntry) value() StorageValue         { return se.data }
func (se AppendStorageEntry) optionalValue() StorageValue {
	se.materializedInPlace()
	return se.data
}

func (se *AppendStorageEntry) materializedInPlace() {
	currentLength := se.currentLength
	if se.materializedLength != nil && *se.materializedLength == currentLength {
		return
	}
	NewStorageAppend(&se.data).ReplaceLength(se.materializedLength, currentLength)
	se.materializedLength = &currentLength
}
