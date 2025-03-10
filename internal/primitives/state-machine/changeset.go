package statemachine

import "github.com/tidwall/btree"

type ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

type ExecutionMode = uint8

const (
	// Executing in client mode: Removal of all transactions possible.
	ExecutionModeClient = iota
	// Executing in runtime mode: Transactions started by the client are protected.
	ExecutionModeRuntime
)

type InnerValue[V any] struct {
	// Current value. None if value has been deleted.
	value V
	// The set of extrinsic indices where the values has been changed.
	extrinsics Extrinsics
}

type DirtyKeysSets[K ordered] []btree.Set[K]

func (dks DirtyKeysSets[K]) Pop() (btree.Set[K], bool) {
	if len(dks) == 0 {
		return btree.Set[K]{}, false
	}

	set := dks[len(dks)-1]
	dks = dks[:len(dks)-1]

	return set, true
}

type Transactions[V any] []InnerValue[V]

type OverlayedChangeSet = OverlayedMap[string, StorageValue]
