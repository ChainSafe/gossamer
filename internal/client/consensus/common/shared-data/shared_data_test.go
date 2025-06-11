package shareddata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSharedData(t *testing.T) {
	// // fn shared_data_locking_works() {
	// t.Run("shared_data_locking_works", func(t *testing.T) {
	// 	// 	const THREADS: u32 = 100;
	// 	// 	let shared_data = SharedData::new(0u32);
	// 	const numThreads = 100
	// 	sharedData := NewSharedDataLock[uint32](0)

	// 	// 	let lock = shared_data.shared_data_locked();
	// 	lock := sharedData.SharedDataLocked()

	// 	// 	for i in 0..THREADS {
	// 	for i := 0; i < numThreads; i++ {
	// 		// 		let data = shared_data.clone();
	// 		data := sharedData
	// 		// 		std::thread::spawn(move || {
	// 		go func(i int) {
	// 			// 			if i % 2 == 1 {
	// 			// 				*data.shared_data() += 1;
	// 			// 			} else {
	// 			// 				let mut lock = data.shared_data_locked().release_mutex();
	// 			// 				// Give the other threads some time to wake up
	// 			// 				std::thread::sleep(std::time::Duration::from_millis(10));
	// 			// 				*lock.upgrade() += 1;
	// 			// 			}
	// 			// 		});
	// 			lock := data.SharedDataLocked()
	// 			defer lock.Unlock()
	// 			released := lock.ReleaseMutex()
	// 			time.Sleep(10 * time.Millisecond)
	// 			ref, unlock := released.Upgrade()
	// 			defer unlock()
	// 			*ref += 1
	// 			// unlock()
	// 			// released.Unlock()
	// 			// lock.Unlock()
	// 		}(i)
	// 	}

	// 	// 	let lock = lock.release_mutex();
	// 	// 	std::thread::sleep(std::time::Duration::from_millis(100));
	// 	// 	drop(lock);
	// 	released := lock.ReleaseMutex()
	// 	// ensure that this is still 0
	// 	time.Sleep(100 * time.Millisecond)
	// 	require.Equal(t, uint32(0), released.sharedData.inner.sharedData)

	// 	released.Unlock()
	// 	lock.Unlock()

	// 	//		while *shared_data.shared_data() < THREADS {
	// 	for {
	// 		//			std::thread::sleep(std::time::Duration::from_millis(100));
	// 		sd := sharedData.SharedData()
	// 		t.Logf("sd: %d", sd)
	// 		if sd >= numThreads {
	// 			break
	// 		}
	// 		time.Sleep(1 * time.Millisecond)
	// 	}

	// })

	t.Run("shared_data_locking_works", func(t *testing.T) {
		const numThreads = 100
		sharedData := NewSharedData[uint32](0)

		ref, unlock := sharedData.DataMut()

		for i := 0; i < numThreads; i++ {
			go func(i int) {
				ref, unlock := sharedData.DataMut()
				defer unlock()
				// Give the other threads some time to wake up
				time.Sleep(10 * time.Millisecond)
				*ref += 1
			}(i)
		}

		time.Sleep(100 * time.Millisecond)
		// ensure that this is still 0
		require.Equal(t, uint32(0), *ref)
		unlock()

		for {
			sd := sharedData.Data()
			t.Logf("sd: %d", sd)
			if sd >= numThreads {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

	})

	t.Run("shared_data_locking_works_with_lock", func(t *testing.T) {
		const numThreads = 100
		sharedData := NewSharedData[uint32](0)

		locked := sharedData.Locked()
		ref := locked.MutRef()

		for i := 0; i < numThreads; i++ {
			go func(i int) {
				locked := sharedData.Locked()
				defer locked.Unlock()
				ref := locked.MutRef()
				// Give the other threads some time to wake up
				time.Sleep(10 * time.Millisecond)
				*ref += 1
			}(i)
		}

		time.Sleep(100 * time.Millisecond)
		// ensure that this is still 0
		require.Equal(t, uint32(0), *ref)
		locked.Unlock()

		for {
			sd := sharedData.Data()
			t.Logf("sd: %d", sd)
			if sd >= numThreads {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

	})
}
