package shareddata

import "sync"

// Some shared data that provides support for locking this shared data for some time.
//
// When working with consensus engines there is often data that needs to be shared between multiple
// parts of the system, like block production and block import. This struct provides an abstraction
// for this shared data in a generic way.
//
// # Deadlock
//
// Be aware that this data structure doesn't give you any guarantees that you can not create a
// deadlock. If you use [SharedDataLocked] by calling [SharedData.Locked] and not call [ShaerdDataLocked.Unlock],
// you will create a deadlock if you try to lock the same [SharedData] again.
type SharedData[T any] struct {
	inner    T
	innerMtx sync.RWMutex
}

// NewSharedData creates a new [SharedData] instance with the provided shared data.
func NewSharedData[T any](sharedData T) *SharedData[T] {
	sd := SharedData[T]{
		inner: sharedData,
	}
	return &sd
}

// Data returns a copy of the shared data.
// It locks the inner mutex for reading, so it is safe to call this method concurrently.
func (sd *SharedData[T]) Data() T {
	sd.innerMtx.RLock()
	defer sd.innerMtx.RUnlock()

	return sd.inner
}

// DataMut returns a mutable reference to the shared data and a function to unlock the inner mutex.
func (sd *SharedData[T]) DataMut() (*T, func()) {
	sd.innerMtx.Lock()
	return &sd.inner, sd.innerMtx.Unlock
}

// Locked returns a [SharedDataLocked] instance that locks the inner mutex for writing.
// [SharedDataLocked.Unlock] must be called to release the lock.
func (sd *SharedData[T]) Locked() SharedDataLocked[T] {
	sd.innerMtx.Lock()
	return SharedDataLocked[T]{sharedData: sd}
}

// SharedDataLocked is referenced to the shared data after calling [SharedData.Locked].
type SharedDataLocked[T any] struct {
	sharedData *SharedData[T]
}

// Data returns the shared data that is locked.
func (sdl SharedDataLocked[T]) Data() T {
	return sdl.sharedData.inner
}

// MutRef returns a mutable reference to the shared data that is locked.
func (sdl SharedDataLocked[T]) MutRef() *T {
	return &sdl.sharedData.inner
}

// Unlock releases the lock on the shared data.
func (sdl SharedDataLocked[T]) Unlock() {
	sdl.sharedData.innerMtx.Unlock()
}
