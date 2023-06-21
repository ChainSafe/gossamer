// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSharedDataGenericReadWrite(t *testing.T) {
	sharedData := NewSharedDataGeneric("hello world")

	go func(sharedData *SharedDataGeneric[string]) {
		sharedData.Write("goodbye")
	}(sharedData)

	data := sharedData.Read()

	require.Equal(t, "hello world", data)

	time.Sleep(3 * time.Second)

	data = sharedData.Read()

	require.Equal(t, "goodbye", data)
}

func TestSharedDataGenericAcquireRelease(t *testing.T) {
	sharedData := NewSharedDataGeneric("hello world")

	sharedData.Acquire()

	go func(sharedData *SharedDataGeneric[string]) {
		sharedData.Acquire()
		sharedData.inner = "goodbye"
		sharedData.Release()
	}(sharedData)

	data := sharedData.inner

	require.Equal(t, "hello world", data)

	sharedData.Release()

	time.Sleep(3 * time.Second)

	data = sharedData.Read()

	require.Equal(t, "goodbye", data)
}

// # use sc_consensus::shared_data::SharedData;
//
// let shared_data = SharedData::new(String::from("hello world"));
//
// let lock = shared_data.shared_data_locked();
//
// let shared_data2 = shared_data.clone();
//
//	let join_handle1 = std::thread::spawn(move || {
//	    // This will need to wait for the outer lock to be released before it can access the data.
//	    shared_data2.shared_data().push_str("1");
//	});
//
// assert_eq!(*lock, "hello world");
//
// // Let us release the mutex, but we still keep it locked.
// // Now we could call `await` for example.
// let mut lock = lock.release_mutex();
//
// let shared_data2 = shared_data.clone();
//
//	let join_handle2 = std::thread::spawn(move || {
//	    shared_data2.shared_data().push_str("2");
//	});
//
// // We still have the lock and can upgrade it to access the data.
// assert_eq!(*lock.upgrade(), "hello world");
// lock.upgrade().push_str("3");
//
// drop(lock);
// join_handle1.join().unwrap();
// join_handle2.join().unwrap();
//
// let data = shared_data.shared_data();
// // As we don't know the order of the threads, we need to check for both combinations
// assert!(*data == "hello world321" || *data == "hello world312");
func TestSharedDataGeneric(t *testing.T) {
	sharedData := NewSharedDataGeneric("hello world")
}
