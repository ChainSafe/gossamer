package shareddata

import "sync"

// // / Created by [`SharedDataLocked::release_mutex`].
// // /
// // / As long as the object isn't dropped, the shared data is locked. It is advised to drop this
// // / object when the shared data doesn't need to be locked anymore. To get access to the shared data
// // / [`Self::upgrade`] is provided.
// // #[must_use = "Shared data will be unlocked on drop!"]
// //
// //	pub struct SharedDataLockedUpgradable<T> {
// //		shared_data: SharedData<T>,
// //	}
// type SharedDataLockedUpgradable[T any] struct {
// 	sharedData *SharedDataLock[T]
// }

// //	impl<T> SharedDataLockedUpgradable<T> {
// //		/// Upgrade to a *real* mutex guard that will give access to the inner data.
// //		///
// //		/// Every call to this function will reaquire the mutex again.
// //		pub fn upgrade(&mut self) -> MappedMutexGuard<T> {
// //			MutexGuard::map(self.shared_data.inner.lock(), |i| &mut i.shared_data)
// //		}
// //	}
// func (sdlu *SharedDataLockedUpgradable[T]) Upgrade() (*T, func()) {
// 	sdlu.sharedData.innerMtx.Lock()
// 	return &sdlu.sharedData.inner.sharedData, func() {
// 		sdlu.sharedData.innerMtx.Unlock()
// 		sdlu.Unlock()
// 	}
// }

// // impl<T> Drop for SharedDataLockedUpgradable<T> {
// // 	fn drop(&mut self) {
// // 		let mut inner = self.shared_data.inner.lock();
// // 		// It should not be locked anymore
// // 		inner.locked = false;

// //			// Notify all waiting threads.
// //			self.shared_data.cond_var.notify_all();
// //		}
// //	}
// func (sdlu *SharedDataLockedUpgradable[T]) Unlock() {
// 	sdlu.sharedData.innerMtx.Lock()
// 	defer sdlu.sharedData.innerMtx.Unlock()
// 	// It should not be locked anymore
// 	inner := &sdlu.sharedData.inner
// 	inner.locked = false

// 	// Notify all waiting threads.
// 	sdlu.sharedData.condVar.Broadcast()
// }

// // / Created by [`SharedData::shared_data_locked`].
// // /
// // / As long as this object isn't dropped, the shared data is held in a mutex guard and the shared
// // / data is tagged as locked. Access to the shared data is provided through
// // / [`Deref`](std::ops::Deref) and [`DerefMut`](std::ops::DerefMut). The trick is to use
// // / [`Self::release_mutex`] to release the mutex, but still keep the shared data locked. This means
// // / every other thread trying to access the shared data in this time will need to wait until this
// // / lock is freed.
// // /
// // / If this object is dropped without calling [`Self::release_mutex`], the lock will be dropped
// // / immediately.
// // #[must_use = "Shared data will be unlocked on drop!"]
// // pub struct SharedDataLocked<'a, T> {
// type SharedDataLocked[T any] struct {
// 	// /// The current active mutex guard holding the inner data.
// 	// inner: MutexGuard<'a, SharedDataInner<T>>,
// 	inner *sharedDataInner[T]
// 	// /// The [`SharedData`] instance that created this instance.
// 	// ///
// 	// /// This instance is only taken on drop or when calling [`Self::release_mutex`].
// 	// shared_data: Option<SharedData<T>>,
// 	sharedData *SharedDataLock[T]
// }

// func (sdl *SharedDataLocked[T]) ReleaseMutex() SharedDataLockedUpgradable[T] {
// 	/// Release the mutex, but keep the shared data locked.
// 	sharedData := sdl.sharedData
// 	if sharedData == nil {
// 		panic("shared data is already released")
// 	}
// 	sdl.sharedData = nil
// 	return SharedDataLockedUpgradable[T]{sharedData: sharedData}
// }

// // impl<'a, T> Drop for SharedDataLocked<'a, T> {
// // 	fn drop(&mut self) {
// // 		if let Some(shared_data) = self.shared_data.take() {
// // 			// If the `shared_data` is still set, it means [`Self::release_mutex`] wasn't
// // 			// called and the lock should be released.
// // 			self.inner.locked = false;

// //				// Notify all waiting threads about the released lock.
// //				shared_data.cond_var.notify_all();
// //			}
// //		}
// //	}
// func (sdl *SharedDataLocked[T]) Unlock() {
// 	if sdl.sharedData != nil {
// 		sharedData := sdl.sharedData
// 		sdl.sharedData = nil
// 		// If the `shared_data` is still set, it means [`Self::release_mutex`] wasn't
// 		// called and the lock should be released.
// 		sdl.inner.locked = false

// 		// Notify all waiting threads about the released lock.
// 		sharedData.condVar.Broadcast()
// 	}
// }

// // / Holds the shared data and if the shared data is currently locked.
// // /
// // / For more information see [`SharedData`].
// // struct SharedDataInner<T> {
// type sharedDataInner[T any] struct {
// 	// /// The actual shared data that is protected here against concurrent access.
// 	// shared_data: T,
// 	sharedData T
// 	// /// Is `shared_data` currently locked and can not be accessed?
// 	// locked: bool,
// 	locked bool
// }

// type SharedDataLock[T any] struct {
// 	inner    sharedDataInner[T]
// 	innerMtx sync.Mutex
// 	condVar  *sync.Cond
// }

// // / Create a new instance of [`SharedData`] to share the given `shared_data`.
// //
// //	pub fn new(shared_data: T) -> Self {
// //		Self {
// //			inner: Arc::new(Mutex::new(SharedDataInner { shared_data, locked: false })),
// //			cond_var: Default::default(),
// //		}
// //	}
// func NewSharedDataLock[T any](sharedData T) *SharedDataLock[T] {
// 	sd := SharedDataLock[T]{
// 		inner: sharedDataInner[T]{
// 			sharedData: sharedData,
// 			locked:     false,
// 		},
// 	}
// 	sd.condVar = sync.NewCond(&sd.innerMtx)
// 	return &sd
// }

// // / Acquire access to the shared data.
// // /
// // / This will give mutable access to the shared data. After the returned mutex guard is dropped,
// // / the shared data is accessible by other threads. So, this function should be used when
// // / reading/writing of the shared data in a local context is required.
// // /
// // / When requiring to lock shared data for some longer time, even with temporarily releasing the
// // / lock, [`Self::shared_data_locked`] should be used.
// // pub fn shared_data(&self) -> MappedMutexGuard<T> {
// // let mut guard = self.inner.lock();
// // while guard.locked {
// // 	self.cond_var.wait(&mut guard);
// // }
// //
// // debug_assert!(!guard.locked);
// //
// // MutexGuard::map(guard, |i| &mut i.shared_data)
// // }

// // 	let mut guard = self.inner.lock();

// // 	while guard.locked {
// // 		self.cond_var.wait(&mut guard);
// // 	}

// // 	debug_assert!(!guard.locked);

// //		MutexGuard::map(guard, |i| &mut i.shared_data)
// //	}
// func (sd *SharedDataLock[T]) SharedData() T {
// 	sd.innerMtx.Lock()
// 	defer sd.innerMtx.Unlock()

// 	for sd.inner.locked {
// 		sd.condVar.Wait()
// 	}
// 	if sd.inner.locked {
// 		panic("shared data is already locked")
// 	}

// 	return sd.inner.sharedData
// }

// // / Acquire access to the shared data and lock it.
// // /
// // / This will give mutable access to the shared data. The returned [`SharedDataLocked`]
// // / provides the function [`SharedDataLocked::release_mutex`] to release the mutex, but
// // / keeping the data locked. This is useful in async contexts for example where the data needs
// // / to be locked, but a mutex guard can not be held.
// // /
// // / For an example see [`SharedData`].
// // pub fn shared_data_locked(&self) -> SharedDataLocked<T> {
// func (sd *SharedDataLock[T]) SharedDataLocked() SharedDataLocked[T] {
// 	// 	let mut guard = self.inner.lock();
// 	sd.innerMtx.Lock()
// 	defer sd.innerMtx.Unlock()

// 	// 	while guard.locked {
// 	// 		self.cond_var.wait(&mut guard);
// 	// 	}
// 	for sd.inner.locked {
// 		sd.condVar.Wait()
// 	}

// 	// 	debug_assert!(!guard.locked);
// 	if sd.inner.locked {
// 		panic("shared data is already locked")
// 	}
// 	// 	guard.locked = true;
// 	sd.inner.locked = true

// 	// SharedDataLocked { inner: guard, shared_data: Some(self.clone()) }
// 	return SharedDataLocked[T]{
// 		inner:      &sd.inner,
// 		sharedData: sd,
// 	}
// }

type SharedData[T any] struct {
	inner    T
	innerMtx sync.RWMutex
}

func NewSharedData[T any](sharedData T) *SharedData[T] {
	sd := SharedData[T]{
		inner: sharedData,
	}
	return &sd
}

func (sd *SharedData[T]) Data() T {
	sd.innerMtx.RLock()
	defer sd.innerMtx.RUnlock()

	return sd.inner
}

func (sd *SharedData[T]) DataMut() (*T, func()) {
	sd.innerMtx.Lock()
	return &sd.inner, sd.innerMtx.Unlock
}

func (sd *SharedData[T]) Locked() SharedDataLocked[T] {
	sd.innerMtx.Lock()
	return SharedDataLocked[T]{sharedData: sd}
}

type SharedDataLocked[T any] struct {
	sharedData *SharedData[T]
}

func (sdl SharedDataLocked[T]) Data() T {
	return sdl.sharedData.inner
}

func (sdl SharedDataLocked[T]) MutRef() *T {
	return &sdl.sharedData.inner
}

func (sdl SharedDataLocked[T]) Unlock() {
	sdl.sharedData.innerMtx.Unlock()
}
