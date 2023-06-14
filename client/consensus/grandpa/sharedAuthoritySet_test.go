// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

type testMutexStruct struct {
	lock sync.Mutex
	num  int
}

type testCondVar struct {
	cond *sync.Cond
	num  int
}

func TestMutex(t *testing.T) {
	testStruct := testMutexStruct{}
	testStruct.lock.Lock()

	go func(testStruct *testMutexStruct) {
		testStruct.num++
	}(&testStruct)

	require.Equal(t, 0, testStruct.num)

	testStruct.lock.Unlock()

	time.Sleep(1 * time.Second)

	require.Equal(t, 1, testStruct.num)
}

func TestCondVar(t *testing.T) {
	lock := sync.Mutex{}
	cond := sync.NewCond(&lock)
	testCond := testCondVar{
		cond: cond,
		num:  0,
	}

	// Locks the mutex
	testCond.cond.L.Lock()

	// Start a separate goroutine to increment the number
	go func(testCond *testCondVar) {
		testCond.num++
		testCond.cond.Signal()
	}(&testCond)

	for testCond.num == 0 {
		// Wait for other routine to awaken this one
		testCond.cond.Wait()
	}

	require.Equal(t, 1, testCond.num)

}

// / # Example
// /
// / ```
// / # use sc_consensus::shared_data::SharedData;
// /
// / let shared_data = SharedData::new(String::from("hello world"));
// /
// / let lock = shared_data.shared_data_locked();
// /
// / let shared_data2 = shared_data.clone();
// / let join_handle1 = std::thread::spawn(move || {
// /     // This will need to wait for the outer lock to be released before it can access the data.
// /     shared_data2.shared_data().push_str("1");
// / });
// /
// / assert_eq!(*lock, "hello world");
// /
// / // Let us release the mutex, but we still keep it locked.
// / // Now we could call `await` for example.
// / let mut lock = lock.release_mutex();
// /
// / let shared_data2 = shared_data.clone();
// / let join_handle2 = std::thread::spawn(move || {
// /     shared_data2.shared_data().push_str("2");
// / });
// /
// / // We still have the lock and can upgrade it to access the data.
// / assert_eq!(*lock.upgrade(), "hello world");
// / lock.upgrade().push_str("3");
// /
// / drop(lock);
// / join_handle1.join().unwrap();
// / join_handle2.join().unwrap();
// /
// / let data = shared_data.shared_data();
// / // As we don't know the order of the threads, we need to check for both combinations
// / assert!(*data == "hello world321" || *data == "hello world312");
// / ```
// /
// / # Deadlock
// /
// / Be aware that this data structure doesn't give you any guarantees that you can not create a
// / deadlock. If you use [`release_mutex`](SharedDataLocked::release_mutex) followed by a call
// / to [`shared_data`](Self::shared_data) in the same thread will make your program dead lock.
// / The same applies when you are using a single threaded executor.
func TestSharedDataInner(t *testing.T) {
	sharedData := SharedDataInner{
		lock:   sync.Mutex{},
		inner:  AuthoritySet{},
		locked: false,
	}

	fmt.Println(sharedData)

}
